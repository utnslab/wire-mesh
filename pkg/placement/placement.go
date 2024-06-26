package placement

import (
	"sync"
	xp "xPlane"
	"xPlane/pkg/placement/smt"

	glog "github.com/golang/glog"
)

type platformInfo struct {
	policies   []xp.Policy
	applGraph  map[string][]string
	services   []string
	hasSidecar []bool
}

type smtOutput struct {
	sat   bool
	s     []string
	impls [][]string
}

func worker(pi platformInfo, target int) smtOutput {
	glog.Info("Running search for target: ", target)
	sat, s, i := smt.OptimizeForTargetDeprecated(pi.policies, pi.applGraph, pi.services, pi.hasSidecar, target)
	return smtOutput{sat, s, i}
}

func runParallelSearches(pi platformInfo, targets []int) (int, smtOutput) {
	numThreads := len(targets)
	outputs := make([]smtOutput, numThreads)

	var wg sync.WaitGroup
	for w := 0; w < numThreads; w++ {
		wg.Add(1)
		w := w
		go func() {
			defer wg.Done()
			outputs[w] = worker(pi, targets[w])
		}()
	}
	wg.Wait()

	// Find the first thread that found a solution.
	for i := 0; i < numThreads; i++ {
		if outputs[i].sat {
			return i, outputs[i]
		}
	}

	return -1, smtOutput{}
}

// Find the optimal placement for the given policies by running search in parallel.
// Requires all dataplane functions to be registered.
func GetPlacementParallel(policies []xp.Policy, applGraph map[string][]string, services []string, hasSidecars []bool, maxThreads int) ([]string, [][]string) {
	pi := platformInfo{policies, applGraph, services, hasSidecars}

	// Get the optimal placement for the given policies.
	var sidecars []string
	var impls [][]string

	// Perform a binary search to find the optimal placement.
	low := 1
	high := len(services)

	for low <= high {
		// Find maxThreads targets to search in parallel.
		var targets []int
		if high-low < maxThreads {
			for i := low; i <= high; i++ {
				targets = append(targets, i)
			}
		} else {
			for i := 0; i < maxThreads; i++ {
				targets = append(targets, low+int((high-low)*i/(maxThreads-1)))
			}
		}
		glog.Info("Running parallel searches for targets: ", targets)

		// Run the searches in parallel.
		i, output := runParallelSearches(pi, targets)

		if i == -1 {
			// No solution found.
			glog.Info("No solution found for targets: ", targets)
			break
		} else {
			// Solution found.
			sidecars = output.s
			impls = output.impls

			// If the first search found a solution. No need to search for lower targets.
			if i == 0 {
				break
			}

			// The first search did not find a solution. Search for lower targets.
			high = targets[i] - 1
			low = targets[i-1] + 1
		}
	}

	// Print the optimal placement.
	glog.Infof("Optimal placement: %d %v", len(sidecars), sidecars)
	glog.Infof("Optimal implementations: %v", impls)

	return sidecars, impls
}

// Find the optimal placement for the given policies. Requires all dataplane functions to be registered.
// Uses the z3 solver's SMT-LIB to find the optimal placement.
func GetPlacement(policies []xp.Policy, applGraph map[string][]string, services []string, sidecarAssignments map[string]int, sidecarCosts []int) (map[string]int, [][]string) {
	// Generate the SMT-LIB file.
	err := smt.GenerateOptimizationFile(policies, applGraph, services, sidecarAssignments, sidecarCosts)
	if err != nil {
		glog.Error("Error generating SMT-LIB file: ", err)
		return nil, nil
	}

	// Run the SMT solver and get the optimal placement for the given policies.
	_, sidecars, impls := smt.RunSolver(services, len(sidecarCosts), len(policies))

	// Print the optimal placement.
	sidecarsMap := make(map[string]int)
	for i, s := range sidecars {
		if s != -1 {
			sidecarsMap[i] = s
		}
	}
	// glog.Infof("Optimal placement: %d %v", len(sidecarsMap), sidecarsMap)
	// glog.Infof("Optimal implementations: %v", impls)

	// Compute cost benefits.
	maxCost := 0
	for _, c := range sidecarCosts {
		if c > maxCost {
			maxCost = c
		}
	}
	maxCost *= len(services)

	// Compute the cost of the optimal placement.
	cost := 0
	instancesUsed := make(map[int]int)
	for _, s := range services {
		sidecar := sidecars[s]
		if sidecar == -1 {
			continue
		}

		cost += sidecarCosts[sidecar]
		instancesUsed[sidecar]++
	}
	glog.Infof("Cost of optimal placement: %d vs max cost: %d", cost, maxCost)
	glog.Info("Instances used: ", instancesUsed)

	return sidecars, impls
}

func GetPlacementBatches(policies []xp.Policy, applGraph map[string][]string, services []string, hasSidecars []bool, maxThreads int, batchSize int) ([]string, [][]string) {
	// Divide the policies into batches.
	var batches [][]xp.Policy
	for i := 0; i < len(policies); i += batchSize {
		end := i + batchSize
		if end > len(policies) {
			end = len(policies)
		}
		batches = append(batches, policies[i:end])
	}

	// Get the optimal placement for the given policies.
	var sidecars []string
	var impls [][]string

	for _, batch := range batches {
		sidecars, impls = GetPlacementParallel(batch, applGraph, services, hasSidecars, maxThreads)

		// Update hasSidecar.
		hasSidecars = make([]bool, len(services))
		for _, s := range sidecars {
			// Find the index of s in services.
			for i, svc := range services {
				if s == svc {
					hasSidecars[i] = true
				}
			}
		}
	}

	return sidecars, impls
}

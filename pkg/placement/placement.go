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
	sat, s, i := smt.OptimizeForTarget(pi.policies, pi.applGraph, pi.services, pi.hasSidecar, target)
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
func GetPlacementParallel(policies []xp.Policy, applGraph map[string][]string, services []string, hasSidecars []bool, maxThreads int) error {
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

	return nil
}

// Find the optimal placement for the given policies. Requires all dataplane functions to be registered.
func GetPlacement(policies []xp.Policy, applGraph map[string][]string, services []string, hasSidecars []bool) error {
	// Get the optimal placement for the given policies.
	var sidecars []string
	var impls [][]string

	// Perform a binary search to find the optimal placement.
	low := 0
	high := len(services)
	for low < high {
		mid := (low + high) / 2
		sat, s, i := smt.OptimizeForTarget(policies, applGraph, services, hasSidecars, mid)
		if sat {
			high = mid
			sidecars = s
			impls = i
		} else {
			low = mid + 1
		}
	}

	// Print the optimal placement.
	glog.Infof("Optimal placement: %d %v", len(sidecars), sidecars)
	glog.Infof("Optimal implementations: %v", impls)

	return nil
}

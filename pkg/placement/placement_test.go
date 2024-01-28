package placement

import (
	"flag"
	"strings"
	"testing"
	"time"

	xp "xPlane"

	glog "github.com/golang/glog"
)

func TestPlacement(t *testing.T) {
	flag.Parse()

	// Create a dummy application graph.
	applGraph := make(map[string][]string)
	applGraph["A"] = []string{"B", "C"}

	// Create a dummy list of services.
	services := []string{"A", "B", "C"}

	// Create a dummy list of policies.
	functions_p1 := []xp.PolicyFunction{
		xp.CreateNewPolicyFunction("set_header", xp.SENDER, []int{0}, true),
		xp.CreateNewPolicyFunction("get_header", xp.SENDER_RECEIVER, []int{0, 1}, false)}

	functions_p2 := []xp.PolicyFunction{
		xp.CreateNewPolicyFunction("set_header", xp.SENDER_RECEIVER, []int{2}, true)}

	policies := []xp.Policy{
		xp.CreatePolicy([]string{"A", "B"}, functions_p1),
		xp.CreatePolicy([]string{"A", "C"}, functions_p2)}

	// Define sidecar costs array.
	sidecarCosts := []int{0, 1, 2}

	// Create an empty map for the initial placement.
	sidecarAssignment := make(map[string]int)

	GetPlacement(policies, applGraph, services, sidecarAssignment, sidecarCosts)
}

var fileName = flag.String("file", "placement_test", "File to read the DAG from")
var generate = flag.Bool("generate", false, "Generate a random DAG")
var fast = flag.Bool("fast", false, "Use the fast solver")
var batch = flag.Bool("batch", false, "Run in batch mode")
var batchSize = flag.Int("batch_size", 4, "Batch size")
var threads = flag.Int("threads", 4, "Number of threads to use")
var testSize = flag.String("size", "medium", "Size of the test instance")
var density = flag.Float64("density", 0.2, "Density of the test instance")

func TestComplete(t *testing.T) {
	flag.Parse()

	var appl Application

	if *generate {
		graphSize := SMALL
		*testSize = strings.ToLower(*testSize)
		if *testSize == "medium" {
			graphSize = MEDIUM
		} else if *testSize == "large" {
			graphSize = LARGE
		}

		// Generate a random DAG.
		applEdges, services := GenerateDAG(*density, graphSize)

		// Generate policies.
		numPolicies := 2 * len(applEdges)
		if *testSize == "medium" {
			numPolicies = 5 * len(applEdges)
		} else if *testSize == "large" {
			numPolicies = 10 * len(applEdges)
		}
		policies := GeneratePolicies(applEdges, numPolicies)

		// Write the policies to a file.
		appl = Application{applEdges, services, policies}
		WriteApplication(appl, *fileName)
	}

	if !*generate {
		// Read Application from file.
		appl = ReadApplication(*fileName)
	}

	// Get the application graph.
	applEdges := appl.applGraph
	services := appl.services
	policies := appl.policies

	numEdges := 0
	for _, edges := range applEdges {
		numEdges += len(edges)
	}

	// Print the testcase size.
	glog.Info("Testcase size: ", len(services), " services (", numEdges, " edges), ", len(policies), " policies")

	// Print the policies.
	// glog.Info("Policies:")
	// for _, p := range policies {
	// 	glog.Info(p)
	// }

	hasSidecar := make([]bool, len(services))
	for i := range hasSidecar {
		hasSidecar[i] = false
	}

	// Call the SMT function.
	start := time.Now()
	if *batch {
		GetPlacementBatches(policies, applEdges, services, hasSidecar, *threads, *batchSize)
	} else {
		if *fast {
			GetPlacementParallel(policies, applEdges, services, hasSidecar, *threads)
		} else {
			// Using 4 sidecars.
			sidecarCosts := []int{10, 8, 4, 2}
			sidecarAssignment := make(map[string]int)
			GetPlacement(policies, applEdges, services, sidecarAssignment, sidecarCosts)
		}
	}

	elapsed := time.Since(start)
	glog.Info("Time: ", elapsed.Milliseconds(), " ms")
}

func TestAdditionalPolicy(t *testing.T) {
	flag.Parse()

	// Read Application from file.
	appl := ReadApplication(*fileName)

	// Get the application graph.
	applEdges := appl.applGraph
	services := appl.services
	policies := appl.policies

	numEdges := 0
	for _, edges := range applEdges {
		numEdges += len(edges)
	}

	// Print the testcase size.
	glog.Info("Testcase size: ", len(services), " services (", numEdges, " edges), ", len(policies), " policies")

	// Print the policies.
	// glog.Info("Policies:")
	// for _, p := range policies {
	// 	glog.Info(p)
	// }

	sidecarAssignment := make(map[string]int)
	sidecarCosts := []int{10, 8, 4, 2}

	// Get the optimal placement for the given policies.
	updatedAssignments, _ := GetPlacement(policies, applEdges, services, sidecarAssignment, sidecarCosts)

	// Update the sidecar assignment.
	for k, v := range updatedAssignments {
		sidecarAssignment[k] = v
	}

	// Generate more policies.
	numPolicies := []int{1, 5, 10, 15, 20}
	times := make([]float64, len(numPolicies))
	for i, num := range numPolicies {
		policies = GeneratePolicies(applEdges, num)

		// Get the optimal placement for the given policies.
		start := time.Now()
		GetPlacement(policies, applEdges, services, sidecarAssignment, sidecarCosts)
		elapsed := time.Since(start)

		times[i] = float64(elapsed.Milliseconds())
	}

	// Print the times.
	glog.Info("Times: ", times)
}

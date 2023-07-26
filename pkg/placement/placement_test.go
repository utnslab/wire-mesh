package placement

import (
	"flag"
	"math/rand"
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
		xp.CreatePolicyFunction("set_header", xp.SENDER, true),
		xp.CreatePolicyFunction("get_header", xp.SENDER_RECEIVER, false)}

	functions_p2 := []xp.PolicyFunction{
		xp.CreatePolicyFunction("set_header", xp.SENDER, true)}

	policies := []xp.Policy{
		xp.CreatePolicy([]string{"A", "B"}, functions_p1),
		xp.CreatePolicy([]string{"A", "C"}, functions_p2)}

	hasSidecar := make([]bool, len(services))
	for i := range hasSidecar {
		hasSidecar[i] = false
	}

	GetPlacement(policies, applGraph, services, hasSidecar)
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
		numPolicies := len(applEdges) + rand.Intn(10)
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
			GetPlacement(policies, applEdges, services, hasSidecar)
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

	hasSidecar := make([]bool, len(services))
	for i := range hasSidecar {
		hasSidecar[i] = false
	}

	// Get the optimal placement for the given policies.
	sidecars, _ := GetPlacementParallel(policies, applEdges, services, hasSidecar, *threads)

	// Update hasSidecar.
	for _, s := range sidecars {
		// Find the index of s in services.
		for i, svc := range services {
			if s == svc {
				hasSidecar[i] = true
			}
		}
	}

	// Generate more policies.
	numPolicies := []int{1, 2, 4, 8, 16}
	times := make([]float64, len(numPolicies))
	for i, num := range numPolicies {
		policies = GeneratePolicies(applEdges, num)

		// Get the optimal placement for the given policies.
		start := time.Now()
		GetPlacementParallel(policies, applEdges, services, hasSidecar, *threads)
		elapsed := time.Since(start)

		times[i] = float64(elapsed.Milliseconds())
	}

	// Print the times.
	glog.Info("Times: ", times)
}

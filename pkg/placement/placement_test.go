package placement

import (
	"flag"
	"math/rand"
	"strings"
	"testing"

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
var threads = flag.Int("threads", 4, "Number of threads to use")
var testSize = flag.String("size", "medium", "Size of the test instance")
var density = flag.Float64("density", 0.2, "Density of the test instance")

func TestComplete(t *testing.T) {
	flag.Parse()

	var appl Application

	if *generate {
		// Parameters.
		density := *density
		maxPathLength := 5

		graphSize := SMALL
		// Convert testSize to lowercase.
		*testSize = strings.ToLower(*testSize)
		if *testSize == "medium" {
			graphSize = MEDIUM
		} else if *testSize == "large" {
			graphSize = LARGE
		}

		// Generate a random DAG.
		applEdges, services := GenerateDAG(density, graphSize)

		// Get a list of all keys in applEdges.
		nonLeafServices := make([]string, 0)
		for k := range applEdges {
			nonLeafServices = append(nonLeafServices, k)
		}

		// Define functions and constraints.
		setHeaderFunc := xp.CreatePolicyFunction("setHeader", xp.SENDER_RECEIVER, false)
		countFunc := xp.CreatePolicyFunction("count", xp.SENDER_RECEIVER, false)
		setDeadlineFunc := xp.CreatePolicyFunction("setDeadline", xp.SENDER, true)
		loadBalanceFunc := xp.CreatePolicyFunction("loadBalance", xp.SENDER, true)

		functions := []xp.PolicyFunction{setHeaderFunc, countFunc, setDeadlineFunc, loadBalanceFunc}

		// Generate policies.
		policies := make([]xp.Policy, 0)
		numPolicies := len(nonLeafServices) + rand.Intn(10)

		for i := 0; i < numPolicies; i++ {
			// Generate a random policy context.
			context := make([]string, 0)

			// Start with a random service from nonLeafServices.
			svc := nonLeafServices[rand.Intn(len(nonLeafServices))]
			context = append(context, svc)

			length := 1
			for {
				// Choose a random edge from svc. If not, end the policy.
				// Our choice of start node ensures that there is at least one edge.
				edges := applEdges[svc]
				if len(edges) > 0 {
					context = append(context, edges[rand.Intn(len(edges))])
					svc = context[len(context)-1]
				} else {
					break
				}

				length += 1
				if length >= maxPathLength {
					break
				}
			}

			// For each service in the context, replace with * based on a probability.
			for j := 0; j < len(context); j++ {
				if rand.Float64() < 0.5 {
					if j != 0 && context[j-1] != "*" {
						context[j] = "*"
					}
				}
			}

			// Choose a random subset of functions.
			numFunctions := 1 + rand.Intn(len(functions))
			policyFunctions := make([]xp.PolicyFunction, 0)
			for j := 0; j < numFunctions; j++ {
				policyFunctions = append(policyFunctions, functions[j])
			}

			// Create the policy.
			policies = append(policies, xp.CreatePolicy(context, policyFunctions))
		}

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
	if *fast {
		GetPlacementParallel(policies, applEdges, services, hasSidecar, *threads)
	} else {
		GetPlacement(policies, applEdges, services, hasSidecar)
	}
}

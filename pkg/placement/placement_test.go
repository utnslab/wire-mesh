package placement

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	xp "xPlane"

	glog "github.com/golang/glog"
)

type TestInstance struct {
	applGraph map[string][]string
	services  []string
	policies  []xp.Policy
}

// Define an enum for small, medium and large testcases.
type TestSize int

const (
	SMALL TestSize = iota
	MEDIUM
	LARGE
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

	GetPlacement(policies, applGraph, services)
}

// Write a TestInstance to the given file.
func writeTestInstance(instance TestInstance, filename string) {
	// Open the file for writing.
	f, err := os.Create(filename)
	if err != nil {
		glog.Fatalf("Failed to open file %s for writing: %v", filename, err)
	}
	defer f.Close()

	// Write the services.
	binary.Write(f, binary.LittleEndian, uint32(len(instance.services)))
	for _, s := range instance.services {
		binary.Write(f, binary.LittleEndian, uint32(len(s)))
		binary.Write(f, binary.LittleEndian, []byte(s))
	}

	// Write the application graph.
	binary.Write(f, binary.LittleEndian, uint32(len(instance.applGraph)))
	for s, edges := range instance.applGraph {
		// Write the service name.
		binary.Write(f, binary.LittleEndian, uint32(len(s)))
		binary.Write(f, binary.LittleEndian, []byte(s))

		// Write the number of edges.
		binary.Write(f, binary.LittleEndian, uint32(len(edges)))

		// Write the edges.
		for _, e := range edges {
			binary.Write(f, binary.LittleEndian, uint32(len(e)))
			binary.Write(f, binary.LittleEndian, []byte(e))
		}
	}

	// Write the policies.
	binary.Write(f, binary.LittleEndian, uint32(len(instance.policies)))
	for _, p := range instance.policies {
		// Write the policy context.
		binary.Write(f, binary.LittleEndian, uint32(len(p.GetContext())))
		for _, c := range p.GetContext() {
			binary.Write(f, binary.LittleEndian, uint32(len(c)))
			binary.Write(f, binary.LittleEndian, []byte(c))
		}

		// Write the policy functions.
		binary.Write(f, binary.LittleEndian, uint32(len(p.GetFunctions())))
		for _, fn := range p.GetFunctions() {
			// Write the function name.
			binary.Write(f, binary.LittleEndian, uint32(len(fn.GetFunctionName())))
			binary.Write(f, binary.LittleEndian, []byte(fn.GetFunctionName()))

			// Write function constraint.
			binary.Write(f, binary.LittleEndian, uint32(fn.GetConstraint()))

			// Write the function mutability.
			binary.Write(f, binary.LittleEndian, fn.GetMutability())
		}
	}
}

// Read a TestInstance from the given file.
func readTestInstance(filename string) TestInstance {
	// Open the file for reading.
	f, err := os.Open(filename)
	if err != nil {
		glog.Fatalf("Failed to open file %s for reading: %v", filename, err)
	}

	// Read the services.
	var numServices uint32
	binary.Read(f, binary.LittleEndian, &numServices)
	services := make([]string, numServices)
	for i := uint32(0); i < numServices; i++ {
		var length uint32
		binary.Read(f, binary.LittleEndian, &length)
		s := make([]byte, length)
		binary.Read(f, binary.LittleEndian, &s)
		services[i] = string(s)
	}

	// Read the application graph.
	applGraph := make(map[string][]string)
	var numKeys uint32
	binary.Read(f, binary.LittleEndian, &numKeys)
	for i := uint32(0); i < numKeys; i++ {
		var length uint32
		binary.Read(f, binary.LittleEndian, &length)
		s := make([]byte, length)
		binary.Read(f, binary.LittleEndian, &s)
		service := string(s)

		var numEdges uint32
		binary.Read(f, binary.LittleEndian, &numEdges)
		edges := make([]string, numEdges)
		for j := uint32(0); j < numEdges; j++ {
			var length uint32
			binary.Read(f, binary.LittleEndian, &length)
			e := make([]byte, length)
			binary.Read(f, binary.LittleEndian, &e)
			edges[j] = string(e)
		}
		applGraph[service] = edges
	}

	// Read the policies.
	var numPolicies uint32
	binary.Read(f, binary.LittleEndian, &numPolicies)
	policies := make([]xp.Policy, numPolicies)
	for i := uint32(0); i < numPolicies; i++ {
		// Read the policy context.
		var contextLength uint32
		binary.Read(f, binary.LittleEndian, &contextLength)
		context := make([]string, contextLength)
		for j := uint32(0); j < contextLength; j++ {
			var length uint32
			binary.Read(f, binary.LittleEndian, &length)
			c := make([]byte, length)
			binary.Read(f, binary.LittleEndian, &c)
			context[j] = string(c)
		}

		// Read the policy functions.
		var numFunctions uint32
		binary.Read(f, binary.LittleEndian, &numFunctions)
		var functions []xp.PolicyFunction
		for j := uint32(0); j < numFunctions; j++ {
			// Read the function name.
			var length uint32
			binary.Read(f, binary.LittleEndian, &length)
			fn := make([]byte, length)
			binary.Read(f, binary.LittleEndian, &fn)

			// Read function constraint.
			var constraint uint32
			binary.Read(f, binary.LittleEndian, &constraint)

			// Read the function mutability.
			var mutability bool
			binary.Read(f, binary.LittleEndian, &mutability)

			function := xp.CreatePolicyFunction(string(fn), xp.ConstraintType(constraint), mutability)
			functions = append(functions, function)
		}
		policies[i] = xp.CreatePolicy(context, functions)
	}

	return TestInstance{services: services, applGraph: applGraph, policies: policies}
}

func generateDAG(density float64, graphSize TestSize) (map[string][]string, []string) {
	// Define application graph.
	applEdges := make(map[string][]string)
	services := make([]string, 0)
	numEdges := 0

	// Generate a DAG with 4-6 tiers.
	tiers := 3
	if graphSize == MEDIUM {
		tiers = tiers + rand.Intn(3)
	} else if graphSize == LARGE {
		tiers = 3*tiers + rand.Intn(12)
	}
	for i := 0; i < tiers; i++ {
		// Generate 5-10 services in each tier.
		new_services := 5
		if graphSize == MEDIUM {
			new_services = new_services + rand.Intn(5)
		} else if graphSize == LARGE {
			new_services = 5*new_services + rand.Intn(10)
		}
		for _, svc := range services {
			for k := 0; k < new_services; k++ {
				// Generate an edge from service j to service k
				if rand.Float64() < density {
					applEdges[svc] = append(applEdges[svc], fmt.Sprintf("svc-%d-%d", i, k))
					numEdges++
				}
			}
		}

		// Add the new services to the list of services.
		for k := 0; k < new_services; k++ {
			services = append(services, fmt.Sprintf("svc-%d-%d", i, k))
		}
	}
	glog.Info("Using a DAG with ", len(services), " services and ", numEdges, " edges")

	return applEdges, services
}

var fileName = flag.String("file", "placement_test", "File to read the DAG from")
var generate = flag.Bool("generate", false, "Generate a random DAG")
var testSize = flag.String("size", "medium", "Size of the test instance")

func TestComplete(t *testing.T) {
	flag.Parse()

	var testInstance TestInstance

	if *generate {
		// Parameters.
		density := 0.2
		maxPathLength := 5
		minPolicies := 5

		graphSize := SMALL
		// Convert testSize to lowercase.
		*testSize = strings.ToLower(*testSize)
		if *testSize == "medium" {
			graphSize = MEDIUM
		} else if *testSize == "large" {
			graphSize = LARGE
		}

		// Generate a random DAG.
		applEdges, services := generateDAG(density, graphSize)

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
		numPolicies := minPolicies + rand.Intn(10)

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
		testInstance = TestInstance{applEdges, services, policies}
		writeTestInstance(testInstance, *fileName)
	}

	if !*generate {
		// Read TestInstance from file.
		testInstance = readTestInstance(*fileName)
	}

	// Get the application graph.
	applEdges := testInstance.applGraph
	services := testInstance.services
	policies := testInstance.policies

	// Print the policies.
	glog.Info("Policies:")
	for _, p := range policies {
		glog.Info(p)
	}

	// Call the SMT function.
	GetPlacement(policies, applEdges, services)
}

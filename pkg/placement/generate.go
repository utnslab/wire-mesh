package placement

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"

	xp "xPlane"

	glog "github.com/golang/glog"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

type Application struct {
	applGraph map[string][]string
	services  []string
	policies  []xp.Policy
}

// Define an enum for small, medium and large testcases.
type GraphSize int

const (
	SMALL GraphSize = iota
	MEDIUM
	LARGE
)

// Write a Application to the given file.
func WriteApplication(instance Application, filename string) {
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

// Read a Application from the given file.
func ReadApplication(filename string) Application {
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

	return Application{services: services, applGraph: applGraph, policies: policies}
}

func GenerateDAG(density float64, graphSize GraphSize) (map[string][]string, []string) {
	// Define application graph.
	applEdges := make(map[string][]string)
	services := make([]string, 0)
	numEdges := 0

	// Generate a DAG with 4-6 tiers.
	tiers := 3
	if graphSize == MEDIUM {
		tiers = tiers + rand.Intn(3)
	} else if graphSize == LARGE {
		tiers = tiers + rand.Intn(6)
	}
	for i := 0; i < tiers; i++ {
		// Generate 5-10 services in each tier.
		new_services := 5
		if graphSize == MEDIUM {
			new_services = new_services + rand.Intn(5)
		} else if graphSize == LARGE {
			new_services = 2*new_services + rand.Intn(5)
		}

		// For each new service, choose a random number of existing services to connect to.
		servicesCopy := make([]string, len(services))
		copy(servicesCopy, services)

		for k := 0; k < new_services; k++ {
			// Shuffle the services.
			rand.Shuffle(len(servicesCopy), func(i, j int) {
				servicesCopy[i], servicesCopy[j] = servicesCopy[j], servicesCopy[i]
			})

			// Choose some number of services to connect to, based on density.
			edges := int(1 + float64(len(servicesCopy))*density)
			if edges > len(servicesCopy) {
				edges = len(servicesCopy)
			}

			// Add the edges.
			for j := 0; j < edges; j++ {
				applEdges[servicesCopy[j]] = append(applEdges[servicesCopy[j]], fmt.Sprintf("svc-%d-%d", i, k))
			}
			numEdges += edges
		}

		// Add the new services to the list of services.
		for k := 0; k < new_services; k++ {
			services = append(services, fmt.Sprintf("svc-%d-%d", i, k))
		}
	}
	glog.Info("Using a DAG with ", len(services), " services and ", numEdges, " edges")

	return applEdges, services
}

func GeneratePolicies(applEdges map[string][]string, numPolicies int) []xp.Policy {
	// Get a list of all keys in applEdges.
	nonLeafServices := make([]string, 0)
	for k := range applEdges {
		nonLeafServices = append(nonLeafServices, k)
	}

	// Define functions and constraints.
	maxPathLength := 5
	setHeaderFunc := xp.CreatePolicyFunction("setHeader", xp.SENDER_RECEIVER, false)
	countFunc := xp.CreatePolicyFunction("count", xp.SENDER_RECEIVER, false)
	setDeadlineFunc := xp.CreatePolicyFunction("setDeadline", xp.SENDER, true)
	loadBalanceFunc := xp.CreatePolicyFunction("loadBalance", xp.SENDER, true)

	functions := []xp.PolicyFunction{setHeaderFunc, countFunc, setDeadlineFunc, loadBalanceFunc}

	policies := make([]xp.Policy, 0)
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

	return policies
}

// Render the application graph in dot format.
func Render(fileName string, outFile string) {
	// Read the application from the given file.
	appl := ReadApplication(fileName)

	// Render the application graph in dot format.
	g := graph.New(graph.StringHash, graph.Directed())

	// Add the services as nodes.
	for _, s := range appl.services {
		g.AddVertex(s)
	}

	// Add the edges.
	for s, edges := range appl.applGraph {
		for _, e := range edges {
			g.AddEdge(s, e)
		}
	}

	// Write the dot output to the given file.
	f, err := os.Create(outFile)
	if err != nil {
		glog.Fatalf("Failed to open file %s for writing: %v", outFile, err)
	}
	defer f.Close()

	draw.DOT(g, f)
}
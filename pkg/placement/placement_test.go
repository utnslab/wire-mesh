package placement

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	xp "xPlane"

	glog "github.com/golang/glog"
)

func constructGraphAndRun(applGraph map[string][]string) (map[string]int, [][]string) {
	// Create a dummy list of services.
	servicesMap := make(map[string]int)
	for k := range applGraph {
		servicesMap[k] = 1
		for _, v := range applGraph[k] {
			servicesMap[v] = 1
		}
	}

	services := make([]string, len(servicesMap))
	i := 0
	for k := range servicesMap {
		services[i] = k
		i++
	}
	print(len(services), services)

	// Create a dummy list of policies.
	functions_p1 := []xp.PolicyFunction{
		xp.CreateNewPolicyFunction("set_header", xp.SENDER, []int{0}, true)}

	policies := make([]xp.Policy, 0)
	for k, arr := range applGraph {
		for _, v := range arr {
			// If v contains "mongo", "memcached", or "redis", then continue.
			// if strings.Contains(v, "mongo") || strings.Contains(v, "memcached") || strings.Contains(v, "redis") {
			// 	continue
			// }
			policies = append(policies, xp.CreatePolicy([]string{k, v}, functions_p1))
		}
	}

	// Define sidecar costs array.
	sidecarCosts := []int{100}

	// Create an empty map for the initial placement.
	sidecarAssignment := make(map[string]int)

	return GetPlacement(policies, applGraph, services, sidecarAssignment, sidecarCosts)
}

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

func TestSocialNetworkPlacement(t *testing.T) {
	flag.Parse()

	// Create a dummy application graph.
	applGraph := make(map[string][]string)
	applGraph["nginx"] = []string{"social-graph", "user", "compose-post", "user-timeline", "home-timeline"}
	applGraph["social-graph"] = []string{"user", "graph-mongo", "graph-redis"}
	applGraph["user"] = []string{"user-memcached", "user-mongo"}
	applGraph["user-timeline"] = []string{"post-storage", "user-redis", "user-mongo"}
	applGraph["compose-post"] = []string{"home-timeline", "media", "text", "post-storage", "user-timeline", "unique-id", "user"}
	applGraph["home-timeline"] = []string{"post-storage", "social-graph"}
	applGraph["text"] = []string{"url", "user-mention"}
	applGraph["url"] = []string{"url-memcached", "url-mongo"}
	applGraph["user-mention"] = []string{"user-mention-mongo", "user-mention-memcached"}
	applGraph["post-storage"] = []string{"post-storage-mongo", "post-storage-redis"}

	// Run for the given application graph.
	constructGraphAndRun(applGraph)
}

func TestHotelReservationPlacement(t *testing.T) {
	flag.Parse()

	// Create a dummy application graph.
	applGraph := make(map[string][]string)
	applGraph["frontend"] = []string{"recommend", "user", "profile", "search", "reserve"}
	applGraph["search"] = []string{"geo", "rate"}
	applGraph["recommend"] = []string{"recommend-mongo"}
	applGraph["reserve"] = []string{"reserve-mongo", "reserve-memc"}
	applGraph["user"] = []string{"user-mongo"}
	applGraph["rate"] = []string{"rate-mongo", "rate-memc"}
	applGraph["geo"] = []string{"geo=mongo"}
	applGraph["profile"] = []string{"profile-mongo", "profile-memc"}

	// Run for the given application graph.
	constructGraphAndRun(applGraph)
}

var fileName = flag.String("file", "placement_test", "File to read the DAG from")
var generate = flag.Bool("generate", false, "Generate a random DAG")
var fast = flag.Bool("fast", false, "Use the fast solver")
var batch = flag.Bool("batch", false, "Run in batch mode")
var batchSize = flag.Int("batch_size", 4, "Batch size")
var threads = flag.Int("threads", 4, "Number of threads to use")
var testSize = flag.String("size", "medium", "Size of the test instance")
var density = flag.Float64("density", 0.2, "Density of the test instance")
var traces = flag.String("traces", "traces", "JSON file to read traces from")

func TestProductionTraces(t *testing.T) {
	flag.Parse()

	// Read the json file at *traces.
	tracesData, err := os.ReadFile(*traces)
	if err != nil {
		glog.Fatal("Error reading traces file: ", err)
	}

	// Parse the json data.
	var data interface{}
	err = json.Unmarshal(tracesData, &data)
	if err != nil {
		glog.Fatal("Error parsing traces data: ", err)
	}

	// Keep track of statistics.
	removed := make([]float32, 0)
	removedHotspots := make([]float32, 0)

	errorGraphs := 0

	allData := data.(map[string]interface{})
	for _, serviceData := range allData {
		// Make application graph.
		applGraph := make(map[string][]string)
		numEdges := make(map[string]int)

		msData := serviceData.(map[string]interface{})
		for svc, data := range msData {
			for k, v := range data.(map[string]interface{}) {
				if k == "num_edges" {
					numEdges[svc] = int(v.(float64))
				} else {
					// Parse v as a list of strings.
					edges := make([]string, 0)
					for _, e := range v.([]interface{}) {
						edges = append(edges, e.(string))
					}
					applGraph[svc] = edges
				}
			}
		}

		// Run for the given application graph.
		sidecars, _ := constructGraphAndRun(applGraph)

		if len(sidecars) == 0 {
			errorGraphs++
			continue
		}

		// Iterate over the sidecars dictionary and find the number of sidecars that are not -1.
		numSidecars := 0
		hotspots := 0
		totalHotspots := 0
		for svc, v := range sidecars {
			if v != -1 {
				numSidecars++
			}

			if numEdges[svc] > 4 {
				if v == -1 {
					hotspots++
				}
				totalHotspots++
			}
		}

		// Fraction of services that have sidecars.
		fraction := 1 - (float32(numSidecars) / float32(len(sidecars)))
		removed = append(removed, fraction)
		if totalHotspots > 0 {
			removedHotspots = append(removedHotspots, float32(hotspots)/float32(totalHotspots))
		}

		// Write removed and removedHotspots to a JSON file.
		removedData := make(map[string]interface{})
		removedData["removed"] = removed
		removedData["removedHotspots"] = removedHotspots

		// Print removed and removedHotspots.
		glog.Info("Removed: ", removed)
		glog.Info("Removed hotspots: ", removedHotspots)

		removedDataBytes, err := json.Marshal(removedData)
		if err != nil {
			glog.Fatal("Error marshalling removed data: ", err)
		}

		err = os.WriteFile("removed.json", removedDataBytes, 0644)
		if err != nil {
			glog.Fatal("Error writing removed data: ", err)
		}

		glog.Info("<==========================================>")
	}

	// Print the statistics.
	glog.Info("Error graphs: ", errorGraphs)
	glog.Info("Fraction of sidecars removed: ", removed)
	glog.Info("Fraction of hotspots removed: ", removedHotspots)
}

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
			numPolicies = 3 * len(applEdges)
		} else if *testSize == "large" {
			numPolicies = 6 * len(applEdges)
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

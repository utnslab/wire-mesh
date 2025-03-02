package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"golang.org/x/exp/slices"

	xp "xPlane"
	"xPlane/pkg/placement"
)

type Application struct {
	applGraph map[string][]string
	services  []string
	policies  []xp.Policy
}

type Interface struct {
	cost int
	egressActions  []string
	ingressActions []string
	noTagActions   []string
}

var tmpl = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html>
<head>
    <script src="https://d3js.org/d3.v6.min.js"></script>
	<script src="https://unpkg.com/d3-graphviz@3.1.0/build/d3-graphviz.js"></script>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; background-color: #f4f4f4; }
        h2 { color: #333; }
        form { margin-bottom: 20px; display: flex; justify-content: center; gap: 10px; }
        .input-container { display: flex; flex-direction: column; align-items: center; width: 30%; padding: 10px }
        textarea { width: 100%; height: 30vh; padding: 10px; border-radius: 5px; border: 1px solid #ccc; }
        button { padding: 10px 20px; border: none; background: #007BFF; color: white; border-radius: 5px; cursor: pointer; margin-top: 10px; }
        button:hover { background: #0056b3; }
        .img { background: white; border: 1px solid #ccc; margin: 20px auto; width: 800px; height: 30vh}
		.img img { width:100%; height:100%; object-fit: scale-down }
		.legend { display: flex; flex-direction: row; gap: 20px; justify-content: center; align-items: center; margin: 10px}
        .legend-item { display: flex; align-items: center; gap: 10px }
        .color-box { width: 30px; height: 20px; border: 1px solid #000 }
        .red { background-color: red; }
        .orange { background-color: orange; }
        .yellow { background-color: yellow; }
        .green { background-color: green; }
    </style>
</head>
<body>
    <h2>Wire Mesh Playground</h2>
    <form id="graphForm">
        <div class="input-container">
            <h3>Microservice Graph</h3>
            <textarea id="graphInput" placeholder='{"nodes": ["A", "B"], "edges": {"A": ["B"]}}'></textarea>
        </div>
        <div class="input-container">
            <h3>Interface</h3>
            <textarea id="interfaceInput"></textarea>
        </div>
        <div class="input-container">
            <h3>Policy</h3>
            <textarea id="policyInput"></textarea>
        </div>
    </form>
    <button type="submit" form="graphForm">Render Graph</button>
	<div class="legend">
        <div class="legend-item"><div class="color-box red"></div> Dataplane 1</div>
        <div class="legend-item"><div class="color-box orange"></div> Dataplane 2</div>
    </div>
	<div class="img">
		<img src="image/tmp.png" alt="Graph visualization" />
	</div>
	<script>
        document.getElementById("graphForm").addEventListener("submit", function(event) {
            event.preventDefault();
            let graphData = JSON.parse(document.getElementById("graphInput").value);
            let interface = document.getElementById("interfaceInput").value;
            let policy = document.getElementById("policyInput").value;

			// Construct the overall input data
			let inputData = {
				"graph": graphData,
				"interface": interface,
				"policy": policy
			};

			// The fetch returns an SVG object in the data. We can append this to the SVG element in the DOM
            fetch("/submit", {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify(inputData)
            })
			.then(response => response.json())
			.then(data => {
				document.querySelector(".img img").src = "image/tmp.png?" + new Date().getTime();
			});
        });
    </script>
</body>
</html>`))

func findMatchingDataplanes(action string, interfaces []Interface) (matchingDataplanes []int, constraint xp.ConstraintType) {
	for i, iface := range interfaces {
		if slices.Contains(iface.egressActions, action) {
			matchingDataplanes = append(matchingDataplanes, i)
			constraint = xp.SENDER
		} else if slices.Contains(iface.ingressActions, action) {
			matchingDataplanes = append(matchingDataplanes, i)
			constraint = xp.RECEIVER
		} else if slices.Contains(iface.noTagActions, action) {
			matchingDataplanes = append(matchingDataplanes, i)
			constraint = xp.SENDER_RECEIVER
		} else {
			continue
		}
	}

	if len(matchingDataplanes) == 0 {
		return []int{}, xp.SENDER_RECEIVER
	}

	return
}

// parseActions parses the actions from the interface string.
// This is only a temporary parsing solution in Golang for the demo -- the actual Copper parser is written in Rust.
func parseActions(input string) (interfaces []Interface) {
	input = strings.TrimSpace(input)
	interfaceStrs := strings.Split(input, "---")

	costRegex := regexp.MustCompile(`cost: ([0-9]+)`)
	actionRegex := regexp.MustCompile(`action ([a-zA-Z0-9_]+)\(.*\)`)

	for _, iface := range interfaceStrs {
		matches := costRegex.FindStringSubmatch(iface)
		cost := 0
		if len(matches) >= 2 {
			if c, err := strconv.Atoi(matches[1]); err == nil {
				cost = c
			}
		}

		egressActions := make([]string, 0)
		ingressActions := make([]string, 0)
		noTagActions := make([]string, 0)

		var currentTag string		
		egTag := "[Egress]"
		inTag := "[Ingress]"

		for _, line := range strings.Split(iface, "\n") {
			line = strings.TrimSpace(line)
			if line == egTag {
				currentTag = "egress"
				continue
			} else if line == inTag {
				currentTag = "ingress"
				continue
			}

			matches := actionRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				actionName := matches[1]
				switch currentTag {
				case "egress":
					egressActions = append(egressActions, actionName)
				case "ingress":
					ingressActions = append(ingressActions, actionName)
				default:
					noTagActions = append(noTagActions, actionName)
				}
			}
		}

		interfaces = append(interfaces, Interface{
			cost: cost,
			egressActions:  egressActions,
			ingressActions: ingressActions,
			noTagActions:   noTagActions,
		})
	}

	return
}

// parsePolicy parses the policy from the policy string.
// This is only a temporary parsing solution in Golang for the demo -- the actual Copper parser is written in Rust.
func parsePolicy(policiesStr string, interfaces []Interface) (policies []xp.Policy, err error) {
	policyStrs := strings.Split(policiesStr, "---")
	contextRegex := regexp.MustCompile(`context \((.*)\)`)
	
	for _, policyStr := range policyStrs {
		// Extract the context from the policy.
		contextMatches := contextRegex.FindStringSubmatch(policyStr)

		// If there is no context, return an error.
		if len(contextMatches) < 2 {
			return []xp.Policy{}, fmt.Errorf("context not found in policy")
		}

		// Split context by '->'
		contextStr := contextMatches[1]
		contextStr = strings.TrimSpace(contextStr)
		if contextStr[0] == '"' {
			contextStr = contextStr[1:]
		}
		if contextStr[len(contextStr)-1] == '"' {
			contextStr = contextStr[:len(contextStr)-1]
		}

		context := strings.Split(contextStr, "->")
		fmt.Printf("Context: %v\n", context)

		// Extract which actions are used in the policy.
		actionRegex := regexp.MustCompile(`([a-zA-Z0-9_]+)\(.*\)`)
		actionMatches := actionRegex.FindAllStringSubmatch(policyStr, -1)
		
		// Extract the action names from the matches, and see if they are in any list of actions.
		policyFunctions := make([]xp.PolicyFunction, 0)
		for _, match := range actionMatches {
			action := match[1]
			matchingDataplanes, constraint := findMatchingDataplanes(action, interfaces)
			if len(matchingDataplanes) == 0 {
				return []xp.Policy{}, fmt.Errorf("action %s not found in any interface", action)
			}

			function := xp.CreateNewPolicyFunction(action, constraint, matchingDataplanes, true)
			policyFunctions = append(policyFunctions, function)
		}

		policy := xp.CreatePolicy(context, policyFunctions)
		policies = append(policies, policy)
	}

	return
}

func renderImg(appl Application, sidecars map[string]int, impls [][]string) error {
	// Render the application graph in dot format.
	g := graph.New(graph.StringHash, graph.Directed())

	// Add the services as nodes.
	colors := []string{"red", "orange", "yellow", "green"}
	for _, s := range appl.services {
		// If the service has a sidecar, color it accordingly.
		sidecar := sidecars[s]
		color := "white"
		if sidecar != -1 {
			color = colors[sidecar]
		}
		
		// Find if the service implements any policy.
		// Iterate over impls, if s is in impls[i], then add i to the set of policies.
		policySet := make([]int, 0)
		for i, impl := range impls {
			if slices.Contains(impl, s) {
				policySet = append(policySet, i+1)
			}
		}
		
		policyStr := ""
		if len(policySet) > 0 {
			for _, p := range policySet {
				policyStr += fmt.Sprintf("P%d ", p)
			}
		}

		g.AddVertex(s, graph.VertexAttribute("style", "filled"), graph.VertexAttribute("fillcolor", color), graph.VertexAttribute("xlabel", policyStr))
	}

	// Add the edges.
	for s, edges := range appl.applGraph {
		for _, e := range edges {
			g.AddEdge(s, e)
		}
	}

	// Write the dot output to a temporary file, tmp.dot.
	f, err := os.Create("tmp.dot")
	if err != nil {
		return err
	}
	defer f.Close()

	draw.DOT(g, f, draw.GraphAttribute("size", "6,4"))

	// Execute the dot command to render the graph as a SVG.
	cmd := exec.Command("dot", "-Tpng", "tmp.dot", "-Gdpi=250", "-o", "image/tmp.png")
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	})

	fs := http.FileServer(http.Dir("image"))
	http.Handle("/image/", http.StripPrefix("/image/", fs))

	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		var inputData struct {
			Graph     struct {
				Nodes []string          `json:"nodes"`
				Edges map[string][]string `json:"edges"`
			} `json:"graph"`
			Interface string `json:"interface"`
			Policy    string `json:"policy"`
		}

		if err := json.NewDecoder(r.Body).Decode(&inputData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		interfaces := parseActions(inputData.Interface)
		fmt.Printf("Interfaces: %v\n", interfaces)

		policies, err := parsePolicy(inputData.Policy, interfaces)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		appl := Application{
			applGraph: inputData.Graph.Edges,
			services: inputData.Graph.Nodes,
			policies: policies,
		}

		fmt.Printf("Application: %v\n", appl.applGraph)
		fmt.Printf("Policies: %v\n", appl.policies)

		// Sidecar costs -- for now, we assume all sidecars have the same cost.
		sidecarCosts := make([]int, 0)
		for _, iface := range interfaces {
			sidecarCosts = append(sidecarCosts, iface.cost)
		}
		
		// Invoke the control plane to find the placements.
		sidecarAssignment := make(map[string]int)
		sidecars, impls := placement.GetPlacement(appl.policies, appl.applGraph, appl.services, sidecarAssignment, sidecarCosts)

		fmt.Printf("Sidecars: %v\n", sidecars)
		fmt.Printf("Implementations: %v\n", impls)

		err = renderImg(appl, sidecars, impls)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with the SVG string in body.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode("OK")
	})

	fmt.Printf("Starting server on :8080\n")
	http.ListenAndServe(":8080", nil)
}

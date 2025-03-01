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
        .img { background: white; border: 1px solid #ccc; margin: 20px auto; width: 800px; height: 40vh}
		.img img { width:100%; height:100%; object-fit: scale-down; }
    </style>
</head>
<body>
    <h2>Wire Mesh Visualization</h2>
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

func parseActions(input string) (egressActions, ingressActions, noTagActions []string) {
	var currentTag string
	
	egTag := "[Egress]"
	inTag := "[Ingress]"

	actionRegex := regexp.MustCompile(`action ([a-zA-Z0-9_]+)\(.*\)`) 

	for _, line := range strings.Split(input, "\n") {
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
	return
}

func parsePolicy(interfaceStr string, policyStr string) (policy xp.Policy, err error) {
	egressActions, ingressActions, noTagActions := parseActions(interfaceStr)

	// Extract the context from the policy.
	contextRegex := regexp.MustCompile(`context \((.*)\)`)
	contextMatches := contextRegex.FindStringSubmatch(policyStr)

	// If there is no context, return an error.
	if len(contextMatches) < 2 {
		return policy, fmt.Errorf("context not found in policy")
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
		if slices.Contains(egressActions, action) {
			function := xp.CreateNewPolicyFunction(action, xp.SENDER, []int{0}, true)
			policyFunctions = append(policyFunctions, function)
		} else if slices.Contains(ingressActions, action) {
			function := xp.CreateNewPolicyFunction(action, xp.RECEIVER, []int{0}, true)
			policyFunctions = append(policyFunctions, function)
		} else if slices.Contains(noTagActions, action) {
			function := xp.CreateNewPolicyFunction(action, xp.SENDER_RECEIVER, []int{0}, true)
			policyFunctions = append(policyFunctions, function)
		} else {
			return policy, fmt.Errorf("action %s not found in any action list", action)
		}
	}
	
	policy = xp.CreatePolicy(context, policyFunctions)
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
		if sidecar != -1 {
			g.AddVertex(s, graph.VertexAttribute("style", "filled"), graph.VertexAttribute("fillcolor", colors[sidecar]))
		} else {
			g.AddVertex(s, graph.VertexAttribute("style", "filled"), graph.VertexAttribute("fillcolor", "white"))
		}
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

		policy, err := parsePolicy(inputData.Interface, inputData.Policy)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		appl := Application{
			applGraph: inputData.Graph.Edges,
			services: inputData.Graph.Nodes,
			policies: []xp.Policy{
				policy,
			},
		}

		fmt.Printf("Application: %v\n", appl.applGraph)
		
		// Invoke the control plane to find the placements.
		sidecarCosts := []int{100}
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

	http.ListenAndServe(":8080", nil)
}

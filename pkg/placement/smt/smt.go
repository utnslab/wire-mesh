package smt

import (
	"fmt"
	"strings"
	"xPlane"

	z3 "xPlane/ext/go-z3"

	"github.com/golang/glog"
	"golang.org/x/exp/slices"
)

// Get a map of all used services to an index in the services array.
func getSvcMapFromList(services []string) map[string]int {
	svcMap := make(map[string]int)
	for i, service := range services {
		svcMap[service] = i
	}

	return svcMap
}

func getPolicyImpls(policyContext []string, applEdges map[string][]string, svcMap map[string]int) ([]int, []int) {
	var penultimateNodes []int
	var lastNodes []int

	if policyContext[len(policyContext)-1] == "*" {
		// There should be a single penultimate node.
		penultimateSvc := policyContext[len(policyContext)-2]
		penultimateNodes = append(penultimateNodes, svcMap[penultimateSvc])

		// All edges from the penultimate node are last nodes.
		for _, svc := range applEdges[penultimateSvc] {
			lastNodes = append(lastNodes, svcMap[svc])
		}
	} else {
		// The last node is the last element of the policy context.
		lastSvc := policyContext[len(policyContext)-1]
		lastNodes = append(lastNodes, svcMap[lastSvc])

		// The penultimate node set will be all the nodes before the last node.
		for svc, edges := range applEdges {
			if slices.Contains(edges, lastSvc) {
				penultimateNodes = append(penultimateNodes, svcMap[svc])
			}
		}
	}

	return penultimateNodes, lastNodes
}

// forwardPolicyContext takes a policy context and a start node,
// and gets all paths that satisfy the policy context.
// If fullExpand is true, then all paths are expanded to the leaf nodes (useful for conflict detection).
// Otherwise, only the paths to the last node are returned.
func forwardPolicyContext(policyContext []string, applEdges map[string][]string, fullExpand bool) [][]string {
	currContextList := [][]string{{policyContext[0]}}
	prevNode := policyContext[0]

	for i := 1; i < len(policyContext); i++ {
		if policyContext[i] != "*" {
			// Add prevNode to every context in currContextList.
			prevNode = policyContext[i]
			for j := 0; j < len(currContextList); j++ {
				currContextList[j] = append(currContextList[j], prevNode)
			}
		} else {
			targetNode := ""
			if i < len(policyContext)-1 {
				targetNode = policyContext[i+1]
			}

			if targetNode != "" {
				bftQueue := [][]string{}
				for _, n := range applEdges[prevNode] {
					bftQueue = append(bftQueue, []string{n})
				}

				// Keep track of paths from previous node to target node.
				newContextList := [][]string{}
				for len(bftQueue) > 0 {
					currPath := bftQueue[0]
					currNode := currPath[len(currPath)-1]

					// Unroll BFS until target node is found or a leaf is met.
					if currNode == targetNode {
						for _, context := range currContextList {
							contextCopy := append([]string{}, context...)
							newContextList = append(newContextList, append(contextCopy, currPath[:len(currPath)-1]...))
						}
					} else if children, ok := applEdges[currNode]; ok {
						for _, n := range children {
							currPathCopy := append([]string{}, currPath...)
							bftQueue = append(bftQueue, append(currPathCopy, n))
						}
					}
					bftQueue = bftQueue[1:]
				}

				currContextList = newContextList
			} else {
				if fullExpand {
					bftQueue := [][]string{}
					for _, n := range applEdges[prevNode] {
						bftQueue = append(bftQueue, []string{n})
					}

					// Keep track of paths from previous node to target node.
					newContextList := [][]string{}
					for len(bftQueue) > 0 {
						currPath := bftQueue[0]
						currNode := currPath[len(currPath)-1]

						// Unroll BFS until a leaf is met.
						if children, ok := applEdges[currNode]; ok {
							for _, n := range children {
								currPathCopy := append([]string{}, currPath...)
								bftQueue = append(bftQueue, append(currPathCopy, n))
							}
						} else {
							// Leaf node found, add to context list.
							for _, context := range currContextList {
								contextCopy := append([]string{}, context...)
								newContextList = append(newContextList, append(contextCopy, currPath...))
							}
						}
						bftQueue = bftQueue[1:]
					}

					currContextList = newContextList
				} else {
					// Add all children of prevNode to every context in currContextList.
					newContextList := [][]string{}
					for _, context := range currContextList {
						for _, n := range applEdges[prevNode] {
							contextCopy := append([]string{}, context...)
							newContextList = append(newContextList, append(contextCopy, n))
						}
					}
					currContextList = newContextList
				}
			}
		}
	}

	return currContextList
}

// backwardPolicyContext takes a policy context and gets all paths
// that satisfy the policy context and end on the given policy_context.
func backwardPolicyContext(targetNode string, applEdges map[string][]string) [][]string {
	parents := make(map[string][]string)
	for n, e := range applEdges {
		for _, c := range e {
			parents[c] = append(parents[c], n)
		}
	}

	backwardBFTQueue := [][]string{{targetNode}}
	contextList := [][]string{}
	for len(backwardBFTQueue) > 0 {
		currPath := backwardBFTQueue[0]
		currNode := currPath[0]
		if len(currPath) > 1 {
			contextList = append(contextList, currPath)
		}

		if children, ok := parents[currNode]; ok {
			for _, n := range children {
				backwardBFTQueue = append(backwardBFTQueue, append([]string{n}, currPath...))
			}
		}
		backwardBFTQueue = backwardBFTQueue[1:]
	}

	return contextList
}

// ExpandPolicyContext expands the policy context to get all possible request contexts.
func ExpandPolicyContext(policyContext []string, applEdges map[string][]string, fullExpand bool) [][]string {
	if policyContext[0] != "*" {
		contextList := forwardPolicyContext(policyContext, applEdges, fullExpand)
		// glog.Info("Expanded policy context: ", policyContext, " to ", contextList)
		return contextList
	} else {
		preContextList := backwardPolicyContext(policyContext[1], applEdges)
		postContextList := forwardPolicyContext(policyContext[1:], applEdges, fullExpand)
		contextList := [][]string{}
		for _, preContext := range preContextList {
			if len(postContextList) > 0 {
				for _, postContext := range postContextList {
					preContextCopy := append([]string{}, preContext...)
					contextList = append(contextList, append(preContextCopy, postContext[1:]...))
				}
			} else {
				contextList = append(contextList, preContext)
			}
		}

		// glog.Info("Expanded policy context: ", policyContext, " to ", contextList)
		return contextList
	}
}

// OptimizeForTarget takes a list of policies, the application graph, a list of all services and a target.
// Returns a boolean indicating whether the optimization was successful, a list of services where the sidecar should be placed,
// and a map of which sidecars implement which policies.
func OptimizeForTarget(policies []xPlane.Policy, applEdges map[string][]string, services []string, target int) (bool, []string, [][]string) {
	// contextToPolicyMap maps request contexts (as string) to a list.
	// The list stores the indexes to the policies in the policies array.
	contextToPolicyMap := make(map[string][]int)

	// Service map is needed to map service names to their index in the z3 variables.
	svcMap := getSvcMapFromList(services)

	// Iterate through all policies, get all request contexts.
	for i, p := range policies {
		reqContexts := ExpandPolicyContext(p.GetContext(), applEdges, false)

		for _, rc := range reqContexts {
			contextStr := strings.Join(rc, ",")
			contextToPolicyMap[contextStr] = append(contextToPolicyMap[contextStr], i)
		}
	}

	// Useful variables.
	numPolicies := len(policies)
	numServices := len(svcMap)
	numContexts := len(contextToPolicyMap)

	// Get all keys from the map.
	allContexts := make([]string, len(contextToPolicyMap))
	i := 0
	for k := range contextToPolicyMap {
		allContexts[i] = k
		i++
	}

	// Define z3 variables.
	config := z3.NewConfig()
	ctx := z3.NewContext(config)
	config.Close()
	defer ctx.Close()

	// Define the "Belong to the policy context" variables.
	B := make([][]*z3.AST, numContexts)
	for i := 0; i < numContexts; i++ {
		B[i] = make([]*z3.AST, numPolicies)
		for j := 0; j < numPolicies; j++ {
			B[i][j] = ctx.Const(ctx.Symbol(fmt.Sprintf("B_%d_%d", i, j)), ctx.BoolSort())
		}
	}

	// Define the "Implements" variables.
	I := make([][]*z3.AST, numServices)
	for m := 0; m < numServices; m++ {
		I[m] = make([]*z3.AST, numPolicies)
		for j := 0; j < numPolicies; j++ {
			I[m][j] = ctx.Const(ctx.Symbol(fmt.Sprintf("I_%d_%d", m, j)), ctx.BoolSort())
		}
	}

	// Define the "Exists" variables.
	X := make([]*z3.AST, numServices)
	for m := 0; m < numServices; m++ {
		X[m] = ctx.Const(ctx.Symbol(fmt.Sprintf("E_%d", m)), ctx.BoolSort())
	}

	// Define the "Executes" variables.
	E := make([][][]*z3.AST, numContexts)
	for i := 0; i < numContexts; i++ {
		E[i] = make([][]*z3.AST, numPolicies)
		for j := 0; j < numPolicies; j++ {
			E[i][j] = make([]*z3.AST, numServices)
			for m := 0; m < numServices; m++ {
				E[i][j][m] = ctx.Const(ctx.Symbol(fmt.Sprintf("E_%d_%d_%d", i, j, m)), ctx.BoolSort())
			}
		}
	}

	// Define the solver
	s := ctx.NewSolver()
	defer s.Close()

	// Add the constraints.
	// Constraint 1 (Belonging) : A policy must only run on a request context that is a subset of the policy context.
	for i := 0; i < numContexts; i++ {
		validPolicies := contextToPolicyMap[allContexts[i]]
		for j := 0; j < numPolicies; j++ {
			if slices.Contains(validPolicies, j) {
				s.Assert(B[i][j])
			} else {
				s.Assert(B[i][j].Not())
			}
		}
	}

	// Constraint 2 : Some node can implement a policy for a particular request context iff the request context belongs to the policy context.
	for i := 0; i < numContexts; i++ {
		for j := 0; j < numPolicies; j++ {
			someNodeImplements := ctx.False()
			for m := 0; m < numServices; m++ {
				someNodeImplements = someNodeImplements.Or(E[i][j][m].And(I[m][j]).And(X[m]))
			}
			s.Assert(someNodeImplements.Iff(B[i][j]))
		}
	}

	// Constraint 3 : A request context can be implemented only by a node on its path.
	for i := 0; i < numContexts; i++ {
		reqContext := strings.Split(allContexts[i], ",")
		for j := 0; j < numPolicies; j++ {
			// Iterate over all services map.
			for svc, m := range svcMap {
				if !slices.Contains(reqContext, svc) {
					s.Assert(E[i][j][m].Not())
				}
			}
		}
	}

	// Constraint 4 : Some policies can be implemented only at sender or receiver.
	for j := 0; j < numPolicies; j++ {
		penultimateNodes, lastNodes := getPolicyImpls(policies[j].GetContext(), applEdges, svcMap)
		// glog.Info("For policy context ", policies[j].GetContext(), " got penultimate nodes: ", penultimateNodes, " and last nodes: ", lastNodes)

		// Either all penultimate nodes implement the policy or all last nodes implement the policy.
		penultimateImplements := ctx.True()
		lastImplements := ctx.True()

		if policies[j].GetConstraint() != xPlane.RECEIVER {
			for _, m := range penultimateNodes {
				penultimateImplements = penultimateImplements.And(I[m][j])
			}
		} else {
			penultimateImplements = ctx.False()
		}

		if policies[j].GetConstraint() != xPlane.SENDER {
			for _, m := range lastNodes {
				lastImplements = lastImplements.And(I[m][j])
			}
		} else {
			lastImplements = ctx.False()
		}

		s.Assert(penultimateImplements.Xor(lastImplements))

		// All other nodes do not implement the policy.
		for m := 0; m < numServices; m++ {
			if policies[j].GetConstraint() == xPlane.SENDER {
				// Sender policy => any node not in penultimate set should not implement the policy.
				if !slices.Contains(penultimateNodes, m) {
					s.Assert(I[m][j].Not())
				}
			} else if policies[j].GetConstraint() == xPlane.RECEIVER {
				// Receiver policy => any node not in lastNodes set should not implement the policy.
				if !slices.Contains(lastNodes, m) {
					s.Assert(I[m][j].Not())
				}
			} else {
				// Except for penultimate and last nodes, no other node should implement the policy.
				if !slices.Contains(penultimateNodes, m) && !slices.Contains(lastNodes, m) {
					s.Assert(I[m][j].Not())
				}
			}
		}
	}

	// Constraint 5 : Atmost one node implements a policy for a request context.
	zero := ctx.Int(0, ctx.IntSort())
	one := ctx.Int(1, ctx.IntSort())
	for i := 0; i < numContexts; i++ {
		for j := 0; j < numPolicies; j++ {
			// Calculate the total number of nodes that implement the policy for the request context.
			totalImplements := ctx.Int(0, ctx.IntSort())
			for m := 0; m < numServices; m++ {
				totalImplements = totalImplements.Add(E[i][j][m].Ite(one, zero))
			}
			s.Assert(B[i][j].Ite(totalImplements.Eq(one), totalImplements.Eq(zero)))
		}
	}

	// Add the objective function.
	targetConst := ctx.Int(target, ctx.IntSort())
	numSidecars := ctx.Int(0, ctx.IntSort())
	for m := 0; m < numServices; m++ {
		numSidecars = numSidecars.Add(X[m].Ite(one, zero))
	}
	s.Assert(numSidecars.Le(targetConst))

	// Check if the constraints are satisfiable.
	glog.Info("Checking if the constraints are satisfiable for target ", target)
	if v := s.Check(); v != z3.True {
		glog.Info("The given constraints are unsolveable")
		return false, nil, nil
	}

	// Get the model.
	glog.Info("Constraints are satisfiable. Getting the model.")
	model := s.Model()
	defer model.Close()

	// Get the values of the X variables.
	sidecars := make([]string, 0)
	for m := 0; m < numServices; m++ {
		xVal := model.Eval(X[m])
		if xVal.String() == "true" {
			// Find the service name from the svcMap map.
			for svc, i := range svcMap {
				if i == m {
					sidecars = append(sidecars, svc)
				}
			}
		}
	}

	// Get the values of the I variables.
	impls := make([][]string, numPolicies)
	for j := 0; j < numPolicies; j++ {
		impls[j] = make([]string, 0)
		for m := 0; m < numServices; m++ {
			iVal := model.Eval(I[m][j])
			xVal := model.Eval(X[m])
			if iVal.String() == "true" && xVal.String() == "true" {
				// Find the service name from the svcMap map.
				for svc, i := range svcMap {
					if i == m {
						glog.Info("Service ", svc, " implements policy ", j)
						impls[j] = append(impls[j], svc)
					}
				}
			}
		}
	}

	return true, sidecars, impls
}

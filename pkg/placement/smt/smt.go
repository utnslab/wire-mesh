package smt

import (
	"fmt"
	"os"
	"os/exec"
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

// OptimizeForTarget takes a list of policies, the application graph, a list of all services,
// list declaring whether a service already has a sidecar and a target (number of changes).
// Returns a boolean indicating whether the optimization was successful, a list of services
// where the sidecar should be placed, and a map of which sidecars implement which policies.
func OptimizeForTarget(policies []xPlane.Policy, applEdges map[string][]string, services []string, hasSidecar []bool, target int) (bool, []string, [][]string) {
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

	glog.Info("All policies expanded")

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

	glog.Info("Defining variables")

	// Define the "Belong to the policy context" variables.
	// B := make([][]*z3.AST, numContexts)
	// for i := 0; i < numContexts; i++ {
	// 	B[i] = make([]*z3.AST, numPolicies)
	// 	for j := 0; j < numPolicies; j++ {
	// 		B[i][j] = ctx.Const(ctx.Symbol(fmt.Sprintf("B_%d_%d", i, j)), ctx.BoolSort())
	// 	}
	// }

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

	glog.Info("Defining constraints")

	// Define the solver
	s := ctx.NewSolver()
	defer s.Close()

	// Add the constraints.
	// Constraint 1 (Belonging) : A policy must only run on a request context that is a subset of the policy context.
	// for i := 0; i < numContexts; i++ {
	// 	validPolicies := contextToPolicyMap[allContexts[i]]
	// 	for j := 0; j < numPolicies; j++ {
	// 		if slices.Contains(validPolicies, j) {
	// 			s.Assert(B[i][j])
	// 		} else {
	// 			s.Assert(B[i][j].Not())
	// 		}
	// 	}
	// }

	// Constraint 2 : Some node can implement a policy for a particular request context iff the request context belongs to the policy context.
	for i := 0; i < numContexts; i++ {
		validPolicies := contextToPolicyMap[allContexts[i]]
		for j := 0; j < numPolicies; j++ {
			someNodeImplements := ctx.False()
			for m := 0; m < numServices; m++ {
				someNodeImplements = someNodeImplements.Or(E[i][j][m].And(I[m][j]).And(X[m]))
			}
			if slices.Contains(validPolicies, j) {
				s.Assert(someNodeImplements)
			} else {
				s.Assert(someNodeImplements.Not())
			}
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
		validPolicies := contextToPolicyMap[allContexts[i]]
		for j := 0; j < numPolicies; j++ {
			// Calculate the total number of nodes that implement the policy for the request context.
			totalImplements := ctx.Int(0, ctx.IntSort())
			for m := 0; m < numServices; m++ {
				totalImplements = totalImplements.Add(E[i][j][m].Ite(one, zero))
			}

			if slices.Contains(validPolicies, j) {
				s.Assert(totalImplements.Eq(one))
			} else {
				s.Assert(totalImplements.Eq(zero))
			}
		}
	}

	// Add the objective function.
	targetConst := ctx.Int(target, ctx.IntSort())
	numChanges := ctx.Int(0, ctx.IntSort())
	for m := 0; m < numServices; m++ {
		if hasSidecar[m] {
			// If sidecar is already there then X[m] = 1 would correspond to no change.
			numChanges = numChanges.Add(X[m].Ite(zero, one))
		} else {
			// If sidecar is not there then X[m] = 1 would correspond to a change.
			numChanges = numChanges.Add(X[m].Ite(one, zero))
		}
	}
	s.Assert(numChanges.Le(targetConst))

	// Check if the constraints are satisfiable.
	glog.Info("Checking if the constraints are satisfiable for target ", target)
	if v := s.Check(); v != z3.True {
		glog.Infof("The given constraints are unsolveable for target %d.", target)
		return false, nil, nil
	}

	// Get the model.
	glog.Infof("Constraints are satisfiable for target %d. Getting the model.", target)
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
						// glog.Info("Service ", svc, " implements policy ", j)
						impls[j] = append(impls[j], svc)
					}
				}
			}
		}
	}

	return true, sidecars, impls
}

// GenerateOptimizationFile takes a list of policies, the application graph, a list of all services,
// list declaring whether a service already has a sidecar and a target (number of changes).
//
// It generates the z3 constraints and the objective function, which can then be used by a z3 solver.
// Overall functionality is similar to the OptimizeForTarget function.
func GenerateOptimizationFile(policies []xPlane.Policy, applEdges map[string][]string, services []string, hasSidecar []bool) {
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

	glog.Info("All policies expanded")

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

	// File to write the variables and constraints to.
	f, err := os.Create("z3_constraints.smt")
	if err != nil {
		glog.Fatal(err)
	}

	// Define z3 variables.
	glog.Info("Defining variables")

	// Define the "Implements" variables.
	I := make([][]string, numServices)
	for m := 0; m < numServices; m++ {
		I[m] = make([]string, numPolicies)
		for j := 0; j < numPolicies; j++ {
			I[m][j] = fmt.Sprintf("I_%d_%d", m, j)
			f.Write([]byte(fmt.Sprintf("(declare-const I_%d_%d Int)\n", m, j)))
			f.Write([]byte(fmt.Sprintf("(assert (or (= I_%d_%d 0) (= I_%d_%d 1)))\n", m, j, m, j)))
		}
	}

	// Define the "Exists" variables.
	X := make([]string, numServices)
	for m := 0; m < numServices; m++ {
		X[m] = fmt.Sprintf("X_%d", m)
		f.Write([]byte(fmt.Sprintf("(declare-const X_%d Int)\n", m)))
		f.Write([]byte(fmt.Sprintf("(assert (or (= X_%d 0) (= X_%d 1)))\n", m, m)))
	}

	// Define the "Executes" variables.
	E := make([][][]string, numContexts)
	for i := 0; i < numContexts; i++ {
		E[i] = make([][]string, numPolicies)
		for j := 0; j < numPolicies; j++ {
			E[i][j] = make([]string, numServices)
			for m := 0; m < numServices; m++ {
				E[i][j][m] = fmt.Sprintf("E_%d_%d_%d", i, j, m)
				f.Write([]byte(fmt.Sprintf("(declare-const E_%d_%d_%d Int)\n", i, j, m)))
				f.Write([]byte(fmt.Sprintf("(assert (or (= E_%d_%d_%d 0) (= E_%d_%d_%d 1)))\n", i, j, m, i, j, m)))
			}
		}
	}

	// Add the constraints.
	glog.Info("Defining constraints")

	// Constraint 2 : Some node can implement a policy for a particular request context iff the request context belongs to the policy context.
	for i := 0; i < numContexts; i++ {
		validPolicies := contextToPolicyMap[allContexts[i]]
		for j := 0; j < numPolicies; j++ {
			constraint_list := make([]string, 0)
			for m := 0; m < numServices; m++ {
				constraint_list = append(constraint_list, fmt.Sprintf("(= 3 (+ %s %s %s))", E[i][j][m], I[m][j], X[m]))
			}

			someNodeImplements := fmt.Sprintf("(or %s)", strings.Join(constraint_list, " "))

			// Write the constraint to the file.
			if slices.Contains(validPolicies, j) {
				f.Write([]byte(fmt.Sprintf("(assert %s)\n", someNodeImplements)))
			} else {
				f.Write([]byte(fmt.Sprintf("(assert (not %s))\n", someNodeImplements)))
			}
		}
	}

	// Constraint 3 : A request context can be implemented only by a node on its path.
	for i := 0; i < numContexts; i++ {
		reqContext := strings.Split(allContexts[i], ",")
		for j := 0; j < numPolicies; j++ {
			// Iterate over all services map.
			for svc, m := range svcMap {
				if !slices.Contains(reqContext, svc) {
					f.Write([]byte(fmt.Sprintf("(assert (= 0 %s))\n", E[i][j][m])))
				}
			}
		}
	}

	// Constraint 4 : Some policies can be implemented only at sender or receiver.
	for j := 0; j < numPolicies; j++ {
		penultimateNodes, lastNodes := getPolicyImpls(policies[j].GetContext(), applEdges, svcMap)
		// glog.Info("For policy context ", policies[j].GetContext(), " got penultimate nodes: ", penultimateNodes, " and last nodes: ", lastNodes)

		// Either all penultimate nodes implement the policy or all last nodes implement the policy.
		penultimateImplements := ""
		lastImplements := ""

		penultimateList := make([]string, 0)
		lastList := make([]string, 0)

		if policies[j].GetConstraint() != xPlane.RECEIVER {
			for _, m := range penultimateNodes {
				penultimateList = append(penultimateList, fmt.Sprintf("(= 1 %s)", I[m][j]))
			}
			penultimateImplements = fmt.Sprintf("(and %s)", strings.Join(penultimateList, " "))
		}

		if policies[j].GetConstraint() != xPlane.SENDER {
			for _, m := range lastNodes {
				lastList = append(lastList, fmt.Sprintf("(= 1 %s)", I[m][j]))
			}
			lastImplements = fmt.Sprintf("(and %s)", strings.Join(lastList, " "))
		}

		// Write the constraint to the file.
		if penultimateImplements == "" {
			f.Write([]byte(fmt.Sprintf("(assert %s)\n", lastImplements)))
		} else if lastImplements == "" {
			f.Write([]byte(fmt.Sprintf("(assert %s)\n", penultimateImplements)))
		} else {
			f.Write([]byte(fmt.Sprintf("(assert (xor %s %s))\n", penultimateImplements, lastImplements)))
		}

		// All other nodes do not implement the policy.
		for m := 0; m < numServices; m++ {
			if policies[j].GetConstraint() == xPlane.SENDER {
				// Sender policy => any node not in penultimate set should not implement the policy.
				if !slices.Contains(penultimateNodes, m) {
					f.Write([]byte(fmt.Sprintf("(assert (= 0 %s))\n", I[m][j])))
				}
			} else if policies[j].GetConstraint() == xPlane.RECEIVER {
				// Receiver policy => any node not in lastNodes set should not implement the policy.
				if !slices.Contains(lastNodes, m) {
					f.Write([]byte(fmt.Sprintf("(assert (= 0 %s))\n", I[m][j])))
				}
			} else {
				// Except for penultimate and last nodes, no other node should implement the policy.
				if !slices.Contains(penultimateNodes, m) && !slices.Contains(lastNodes, m) {
					f.Write([]byte(fmt.Sprintf("(assert (= 0 %s))\n", I[m][j])))
				}
			}
		}
	}

	// Constraint 5 : Atmost one node implements a policy for a request context.
	for i := 0; i < numContexts; i++ {
		validPolicies := contextToPolicyMap[allContexts[i]]
		for j := 0; j < numPolicies; j++ {
			// Calculate the total number of nodes that implement the policy for the request context.
			totalImplementsList := make([]string, 0)

			for m := 0; m < numServices; m++ {
				totalImplementsList = append(totalImplementsList, E[i][j][m])
			}

			totalImplements := fmt.Sprintf("(+ %s)", strings.Join(totalImplementsList, " "))
			if slices.Contains(validPolicies, j) {
				f.Write([]byte(fmt.Sprintf("(assert (= 1 %s))\n", totalImplements)))
			} else {
				f.Write([]byte(fmt.Sprintf("(assert (= 0 %s))\n", totalImplements)))
			}
		}
	}

	// Add the objective function.
	numChangesList := make([]string, 0)
	for m := 0; m < numServices; m++ {
		if hasSidecar[m] {
			// If sidecar is already there then X[m] = 1 would correspond to no change.
			numChangesList = append(numChangesList, fmt.Sprintf("(- %s 1)", X[m]))
		} else {
			// If sidecar is not there then X[m] = 1 would correspond to a change.
			numChangesList = append(numChangesList, X[m])
		}
	}
	numChanges := fmt.Sprintf("(+ %s)", strings.Join(numChangesList, " "))
	f.Write([]byte(fmt.Sprintf("(minimize %s)\n", numChanges)))

	// Add instructions for the z3 solver.
	f.Write([]byte("(check-sat)\n"))

	// Get the values of I and X variables.
	for m := 0; m < numServices; m++ {
		f.Write([]byte(fmt.Sprintf("(get-value (%s))\n", X[m])))
	}

	for m := 0; m < numServices; m++ {
		for j := 0; j < numPolicies; j++ {
			f.Write([]byte(fmt.Sprintf("(get-value (%s))\n", I[m][j])))
		}
	}

	// Close the file.
	f.Close()
}

// Runs the z3 solver on the generated file and returns the output.
func RunSolver(services []string, numPolicies int) (bool, []string, [][]string) {
	// Use the z3 command line tool to run the solver.
	cmd := exec.Command("z3", "z3_constraints.smt")
	out, err := cmd.CombinedOutput()
	if err != nil {
		glog.Error("Error running z3 solver: ", err)
	}

	// Parse the output of the solver.
	solverOutput := string(out)

	// Split the output of the solver by new line.
	solverOutputLines := strings.Split(solverOutput, "\n")

	// 1st line is sat/unsat.
	if solverOutputLines[0] == "unsat" {
		return false, nil, nil
	}

	// Parse solverOutput to get the values of X and I variables.
	sidecars := make([]string, 0)
	impls := make([][]string, numPolicies)
	X := make([]string, len(services))

	// Get the values of X variables.
	for m := 0; m < len(services); m++ {
		// The value of X[m] is in the form (X_m value).
		// Split the line by space and get the value.
		line := strings.Split(solverOutputLines[m+1], " ")
		X[m] = line[1][:len(line[1])-2]
		if X[m] == "1" {
			sidecars = append(sidecars, services[m])
		}
	}

	// Get the values of I variables.
	for m := 0; m < len(services); m++ {
		for j := 0; j < numPolicies; j++ {
			// The value of I[m][j] is in the form (I_m_j value).
			// Split the line by space and get the value.
			line := strings.Split(solverOutputLines[len(services)+1+m*numPolicies+j], " ")
			iVal := line[1][:len(line[1])-2]
			if iVal == "1" && X[m] == "1" {
				impls[j] = append(impls[j], services[m])
			}
		}
	}

	return true, sidecars, impls
}

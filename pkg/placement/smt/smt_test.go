package smt

import (
	"flag"
	"testing"
	"xPlane/pkg/placement"

	"github.com/golang/glog"
)

func TestBasic(t *testing.T) {
	flag.Parse()

	services := []string{"A", "B", "C", "D", "E", "F", "G"}
	allServices := make(map[string]int)
	for i, service := range services {
		allServices[service] = i
	}

	// Define application graph.
	applEdges := make(map[string][]string)
	applEdges["A"] = []string{"B", "C"}
	applEdges["B"] = []string{"E"}
	applEdges["C"] = []string{"D"}
	applEdges["D"] = []string{"E"}
	applEdges["E"] = []string{"F", "G"}

	// Define functions and constraints.
	setHeaderFunc := placement.CreatePolicyFunction("setHeader", placement.SENDER_RECEIVER)
	countFunc := placement.CreatePolicyFunction("count", placement.SENDER_RECEIVER)
	setDeadlineFunc := placement.CreatePolicyFunction("setDeadline", placement.SENDER)

	// Define policies.
	policies := []placement.Policy{
		placement.CreatePolicy([]string{"A", "*"}, []placement.PolicyFunction{setHeaderFunc}),
		placement.CreatePolicy([]string{"*", "F"}, []placement.PolicyFunction{countFunc}),
		placement.CreatePolicy([]string{"A", "*", "E", "*"}, []placement.PolicyFunction{setDeadlineFunc}),
	}

	// Call the SMT function.
	sidecars, placements := optimizeForTarget(policies, applEdges, allServices, 3)
	glog.Infof("Services with sidecars: %v", sidecars)
	glog.Infof("Placements: %v", placements)
}

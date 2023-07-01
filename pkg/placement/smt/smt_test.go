package smt

import (
	"flag"
	"testing"
	"xPlane"

	"github.com/golang/glog"
)

func TestBasic(t *testing.T) {
	flag.Parse()

	services := []string{"A", "B", "C", "D", "E", "F", "G"}

	// Define application graph.
	applEdges := make(map[string][]string)
	applEdges["A"] = []string{"B", "C"}
	applEdges["B"] = []string{"E"}
	applEdges["C"] = []string{"D"}
	applEdges["D"] = []string{"E"}
	applEdges["E"] = []string{"F", "G"}

	// Define functions and constraints.
	setHeaderFunc := xPlane.CreatePolicyFunction("setHeader", xPlane.SENDER_RECEIVER, false)
	countFunc := xPlane.CreatePolicyFunction("count", xPlane.SENDER_RECEIVER, false)
	setDeadlineFunc := xPlane.CreatePolicyFunction("setDeadline", xPlane.SENDER, false)

	// Define policies.
	policies := []xPlane.Policy{
		xPlane.CreatePolicy([]string{"A", "*"}, []xPlane.PolicyFunction{setHeaderFunc}),
		xPlane.CreatePolicy([]string{"*", "F"}, []xPlane.PolicyFunction{countFunc}),
		xPlane.CreatePolicy([]string{"A", "*", "E", "*"}, []xPlane.PolicyFunction{setDeadlineFunc}),
	}

	// Call the SMT function.
	sat, sidecars, placements := OptimizeForTarget(policies, applEdges, services, 3)
	if !sat {
		glog.Infof("No solution found.")
		return
	}

	glog.Infof("Services with sidecars: %v", sidecars)
	glog.Infof("Placements: %v", placements)
}

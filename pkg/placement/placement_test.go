package placement

import (
	"flag"
	"testing"

	xp "xPlane"
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

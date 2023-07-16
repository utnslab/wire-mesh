package conflict

import (
	"flag"
	"testing"

	xp "xPlane"
)

func TestConflicts(t *testing.T) {
	flag.Parse()

	// Create a dummy application graph.
	applGraph := make(map[string][]string)
	applGraph["A"] = []string{"B", "C"}
	applGraph["B"] = []string{"C"}

	// Create a dummy list of policies.
	functions_p1 := []xp.PolicyFunction{
		xp.CreatePolicyFunction("set_header", xp.SENDER, true),
		xp.CreatePolicyFunction("get_header", xp.SENDER_RECEIVER, false)}

	functions_p2 := []xp.PolicyFunction{
		xp.CreatePolicyFunction("set_header", xp.SENDER, true)}

	policies := []xp.Policy{
		xp.CreatePolicy([]string{"A", "*"}, functions_p1),
		xp.CreatePolicy([]string{"A", "C"}, functions_p2)}

	// Now define a new policy that conflicts with the first one.
	functions_p3 := []xp.PolicyFunction{
		xp.CreatePolicyFunction("set_header", xp.SENDER, true)}

	newPolicy := xp.CreatePolicy([]string{"*", "B", "C"}, functions_p3)

	// Check if the new policy conflicts with the existing ones.
	conflicts := FindConflictingPolicies(policies, newPolicy, applGraph)
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflicting policy, got %d", len(conflicts))
	}
}

package conflict

import (
	"strings"
	xp "xPlane"
	"xPlane/pkg/placement/smt"
)

func overlappingContext(policy1 xp.Policy, policy2 xp.Policy, applGraph map[string][]string) bool {
	// Enumerate all possible contexts for policy1.
	allContexts1 := smt.ExpandPolicyContext(policy1.GetContext(), applGraph)
	allContexts2 := smt.ExpandPolicyContext(policy2.GetContext(), applGraph)

	// Check if any of the contexts is a subset of the other.
	// TODO: This might be a very inefficient way of doing this.
	for _, context1 := range allContexts1 {
		for _, context2 := range allContexts2 {
			strRepr1 := strings.Join(context1, ",")
			strRepr2 := strings.Join(context2, ",")

			if len(context1) > len(context2) {
				// Check if context2 is a subset of context1.
				if strings.Contains(strRepr1, strRepr2) {
					return true
				}
			} else {
				// Check if context1 is a subset of context2.
				if strings.Contains(strRepr2, strRepr1) {
					return true
				}
			}
		}
	}

	return false
}

// Find conflicting policies given a set of already submitted policies,
// and a new policy.
func FindConflictingPolicies(policies []xp.Policy, newPolicy xp.Policy, applGraph map[string][]string) []xp.Policy {
	var conflictingPolicies []xp.Policy

	for _, policy := range policies {
		// A policy could be conflicting if it has overlapping context.
		if overlappingContext(policy, newPolicy, applGraph) {
			// Check if both policies mutate the CNO.
			if policy.ExistsMutableFunction() && newPolicy.ExistsMutableFunction() {
				conflictingPolicies = append(conflictingPolicies, policy)
			}
		}
	}

	return conflictingPolicies
}

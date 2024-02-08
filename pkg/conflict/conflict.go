package conflict

import (
	"strings"
	xp "xPlane"
	"xPlane/pkg/placement/smt"
)

// Check if the policy has overlapping context with the given set of contexts.
func overlappingContext(policy xp.Policy, contexts [][]string, applGraph map[string][]string) bool {
	// Enumerate all possible contexts for policy1.
	allContexts := smt.ExpandPolicyContextDeprecated(policy.GetContext(), applGraph, true)

	// Check if any of the contexts is a subset of the other.
	// TODO: This might be a very inefficient way of doing this.
	for _, context1 := range contexts {
		for _, context2 := range allContexts {
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

	// Get all contexts for the new policy.
	newPolicyContexts := smt.ExpandPolicyContextDeprecated(newPolicy.GetContext(), applGraph, true)

	for _, policy := range policies {
		// A policy could be conflicting if it has overlapping context.
		if overlappingContext(policy, newPolicyContexts, applGraph) {
			// Check if both policies mutate the CNO.
			if policy.ExistsMutableFunction() && newPolicy.ExistsMutableFunction() {
				conflictingPolicies = append(conflictingPolicies, policy)
			}
		}
	}

	return conflictingPolicies
}

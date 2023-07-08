package placement

import (
	"flag"
	"testing"
)

func TestPlacement(t *testing.T) {
	flag.Parse()

	// Testing purposes.
	po := CreatePolicyOptimizer("../../examples", map[string][]string{"A": []string{"B", "C"}})

	po.GetPlacement([]string{"dummy_policy.json"})
}

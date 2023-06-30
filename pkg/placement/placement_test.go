package placement

import (
	"flag"
	"testing"
)

func TestPlacement(t *testing.T) {
	flag.Parse()

	// Testing purposes.
	po := PolicyOptimizer{
		jsonDir: "../../m4language/target/debug/build",
	}
	po.GetPlacement([]string{"ast.json"})
}

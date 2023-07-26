package placement

import (
	"flag"
	"testing"
)

var renderFile = flag.String("file", "placement_test", "File to read the DAG from")
var outFile = flag.String("out", "placement_test.gv", "File to write the dot output to")

func TestRender(t *testing.T) {
	flag.Parse()

	Render(*renderFile, *outFile)
}

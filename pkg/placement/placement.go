package placement

import (
	xp "xPlane"
	"xPlane/pkg/placement/smt"

	glog "github.com/golang/glog"
)

// Find the optimal placement for the given policies. Requires all dataplane functions to be registered.
func GetPlacement(policies []xp.Policy, applGraph map[string][]string, services []string) error {
	// Get the optimal placement for the given policies.
	var sidecars []string
	var impls [][]string

	// Perform a binary search to find the optimal placement.
	low := 0
	high := len(services)
	for low < high {
		mid := (low + high) / 2
		sat, s, i := smt.OptimizeForTarget(policies, applGraph, services, mid)
		if sat {
			high = mid
			sidecars = s
			impls = i
		} else {
			low = mid + 1
		}
	}

	// Print the optimal placement.
	glog.Infof("Optimal placement: %v", sidecars)
	glog.Infof("Optimal implementations: %v", impls)

	return nil
}

package placement

import (
	"os"
	"path"
	"strings"
	xp "xPlane"
	"xPlane/pkg/placement/smt"

	"github.com/buger/jsonparser"
	glog "github.com/golang/glog"
	"golang.org/x/exp/slices"
)

// PolicyOptimizer is the struct that contains the information needed to optimize a policy.
type PolicyOptimizer struct {
	// The directory from where to read the json policy files.
	jsonDir string

	// Microservice application graph.
	services  []string
	applGraph map[string][]string

	// Registry of all available dataplane functions.
	// Map from "dataplane path" to a map of {function name->Function} supported by the dataplane.
	// dataplane path must be "<dataplane name>.m4.json"
	functionsRegistry map[string]map[string]xp.PolicyFunction
}

func (po *PolicyOptimizer) fillServicesFromGraph() {
	for service := range po.applGraph {
		po.services = append(po.services, service)

		// Also add the services that are connected to this service.
		for _, svc := range po.applGraph[service] {
			if !slices.Contains(po.services, svc) {
				po.services = append(po.services, svc)
			}
		}
	}
}

// Add a new dataplane to the functionsRegistry map. Takes the dataplane json file name as input.
// NOTE: Currently requires all functions of a dataplane to be unique.
func (po *PolicyOptimizer) RegisterDataplane(dataplaneJson string) error {
	// Read the json file from the jsonDir.
	b, err := os.ReadFile(path.Join(po.jsonDir, dataplaneJson))
	if err != nil {
		glog.Errorf("Error reading file %s: %v", dataplaneJson, err)
		return err
	}

	// Get the functions from the json file.
	groupObjects, _, _, err := jsonparser.Get(b, "groups")
	if err != nil {
		glog.Errorf("Error getting functions from json bytes: %v", err)
		return err
	}

	// Iterate over the objects in the groups array and get the one which has a CnoInterface.
	var cnoObjects [][]byte
	jsonparser.ArrayEach(groupObjects, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		// Get the CnoInterface from the object.
		cnoObject, _, _, e := jsonparser.Get(value, "inner", "Specification", "CnoInterface")
		if e != nil {
			glog.Errorf("No CnoInterface in object: %s", value)
		}
		cnoObjects = append(cnoObjects, cnoObject)
	})

	// Iterate over the CnoInterfaces and get the functions.
	var functions map[string]xp.PolicyFunction
	for _, cnoObject := range cnoObjects {
		// Get the functions from the CnoInterface.
		functionsArray, _, _, err := jsonparser.Get(cnoObject, "fields")
		if err != nil {
			glog.Errorf("No functions in CnoInterface: %s", cnoObject)
		}

		// Iterate over the functions array and add functions to the `functions` array.
		jsonparser.ArrayEach(functionsArray, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			// Get the function from the object.
			functionName, _, _, e := jsonparser.Get(value, "Action", "name", "name")
			if e != nil {
				glog.Errorf("Could not find function name in: %s", value)
			}

			// Get the constraint from the object.
			constraintObj, _, _, e := jsonparser.Get(value, "Action", "type_", "Function", "[0]", "self_")
			if e != nil {
				glog.Errorf("Could not find function constraint in: %s", value)
			}

			if constraintObj == nil {
				glog.Info("No constraint found for function: %s", functionName)

				pf := xp.CreatePolicyFunction(string(functionName), xp.SENDER_RECEIVER, false)
				functions[string(functionName)] = pf
			} else {
				// Get the placement constraint from the object.
				pConstraint, _, _, e := jsonparser.Get(constraintObj, "placement")
				if e != nil {
					glog.Errorf("Could not find function placement constraint in: %s", value)
				}

				placement := xp.SENDER_RECEIVER
				if string(pConstraint) == "In" {
					placement = xp.RECEIVER
				} else if string(pConstraint) == "Out" {
					placement = xp.SENDER
				}

				// Get the mutable constraint from the object.
				mConstraint, _, _, e := jsonparser.Get(constraintObj, "mutability")
				if e != nil {
					glog.Errorf("Could not find function mutable constraint in: %s", value)
				}

				pf := xp.CreatePolicyFunction(string(functionName), placement, string(mConstraint) == "Mut")
				functions[string(functionName)] = pf
			}
		})
	}

	// Add the functions to the functionsRegistry map.
	po.functionsRegistry[dataplaneJson] = functions

	return nil
}

// Parse a byte array of json file to get the policy struct.
// Requires the dataplane json file to be named as `<filename>.m4.json`.
func (po *PolicyOptimizer) ParsePolicy(b []byte) xp.Policy {
	// Get the context from the json file.
	matchesArray, _, _, err := jsonparser.Get(b, "groups", "[0]", "inner", "Policy", "matches")
	if err != nil {
		glog.Errorf("Error getting context from json bytes: %v", err)
	}

	// Iterate over the objects in the matches array and get the one which has the context.
	var contextObject []byte
	found := false
	jsonparser.ArrayEach(matchesArray, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		// Get the context from the object.
		object, _, _, e := jsonparser.Get(value, "Context")
		if e == nil {
			// TODO: Assumption is that there is only one context object.
			contextObject = object
			found = true
		}
	})

	if !found {
		glog.Errorf("No context found in policy: %s", matchesArray)
	}

	// Get the list of endpoints from the context object.
	endpointsArray, _, _, err := jsonparser.Get(contextObject, "blocks")
	if err != nil {
		glog.Errorf("No blocks in context: %s", contextObject)
	}

	// Iterate over the endpoints array and get the endpoints.
	var context []string
	jsonparser.ArrayEach(endpointsArray, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		// Get the endpoint from the object.
		endpointBlock, _, _, e := jsonparser.Get(value, "inner", "Endpoints")
		if e != nil {
			glog.Errorf("No endpoints in object: %s", value)
		}

		// Iterate over the endpoints array and get the endpoint.
		var serviceSet []string
		jsonparser.ArrayEach(endpointBlock, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			endpoint, _, _, e := jsonparser.Get(value, "name")
			if e != nil {
				glog.Errorf("Service name missing in object: %s", value)
			}
			serviceSet = append(serviceSet, string(endpoint))
		})

		// Append the service set to the context.
		if len(serviceSet) == 1 {
			context = append(context, serviceSet[0])
		} else {
			context = append(context, "["+strings.Join(serviceSet, ",")+"]")
		}
	})
	glog.Infof("Context: %s", context)

	// Get the functions from the json file.
	// NOTE: Currently assumes only a single import.
	pathObj, _, _, err := jsonparser.Get(b, "imports", "[0]", "path")
	if err != nil {
		glog.Errorf("No dataplane name in json bytes: %v", err)
	}
	dataplaneName := string(pathObj) + ".json"

	// Get the list of functions being used in the policy.
	functionsList, _, _, err := jsonparser.Get(b, "groups", "[0]", "inner", "Policy", "used_abstract_fields")
	if err != nil {
		glog.Errorf("Functions list not found in json bytes: %v", err)
	}

	// Iterate over the functions list and get the functions.
	var functions []xp.PolicyFunction
	jsonparser.ArrayEach(functionsList, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		// Get the function from the object.
		jsonparser.ArrayEach(value, func(valueInner []byte, dataTypeInner jsonparser.ValueType, offsetInner int, errInner error) {
			obj, _, _, e := jsonparser.Get(valueInner, "set")
			if e == nil {
				functionName, _, _, e := jsonparser.Get(obj, "[0]")
				if e != nil {
					glog.Errorf("Function name missing in object: %s", valueInner)
				}

				// Get the function from the functionsRegistry.
				pf := po.functionsRegistry[dataplaneName][string(functionName)]
				functions = append(functions, pf)
			}
		})
	})

	return xp.CreatePolicy(context, functions)
}

// Find the optimal placement for the given policies. Requires all dataplane functions to be registered.
func (po *PolicyOptimizer) GetPlacement(policyJsons []string) error {
	// Iterate over the policies.
	var policies []xp.Policy
	for _, pJson := range policyJsons {
		// Read the json file from the jsonDir.
		b, err := os.ReadFile(path.Join(po.jsonDir, pJson))
		if err != nil {
			glog.Errorf("Error reading file %s: %v", pJson, err)
			return err
		}

		// Parse the json file to get the policy struct.
		p := po.ParsePolicy(b)
		policies = append(policies, p)
	}

	// Get the optimal placement for the given policies.
	var sidecars []string
	var impls [][]string

	// Perform a binary search to find the optimal placement.
	low := 0
	high := len(po.services)
	for low < high {
		mid := (low + high) / 2
		sat, s, i := smt.OptimizeForTarget(policies, po.applGraph, po.services, mid)
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

func CreatePolicyOptimizer(jsonDir string, applGraph map[string][]string) *PolicyOptimizer {
	po := PolicyOptimizer{
		jsonDir:           jsonDir,
		functionsRegistry: make(map[string]map[string]xp.PolicyFunction),
		applGraph:         applGraph,
	}

	// Construct the list of services.
	po.fillServicesFromGraph()

	return &po
}

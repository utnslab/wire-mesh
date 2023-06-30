package placement

import (
	"os"
	"path"

	"github.com/buger/jsonparser"
	glog "github.com/golang/glog"
)

type ConstraintType int

const (
	SENDER ConstraintType = iota
	RECEIVER
	SENDER_RECEIVER
)

type PolicyFunction struct {
	functionName string
	constraint   ConstraintType
}

type Policy struct {
	context   []string
	placement string
	functions []PolicyFunction
}

// PolicyOptimizer is the struct that contains the information needed to optimize a policy.
type PolicyOptimizer struct {
	jsonDir   string
	applGraph map[string][]string
}

// Accessor methods for PolicyFunction struct.
func (pf *PolicyFunction) GetFunctionName() string {
	return pf.functionName
}

func (pf *PolicyFunction) GetConstraint() ConstraintType {
	return pf.constraint
}

func CreatePolicyFunction(functionName string, constraint ConstraintType) PolicyFunction {
	return PolicyFunction{
		functionName: functionName,
		constraint:   constraint,
	}
}

// Accessor methods for Policy struct.
func (p *Policy) GetContext() []string {
	return p.context
}

func (p *Policy) GetPlacement() string {
	return p.placement
}

func (p *Policy) SetPlacement(placement string) {
	p.placement = placement
}

func (p *Policy) GetFunctions() []PolicyFunction {
	return p.functions
}

func CreatePolicy(context []string, functions []PolicyFunction) Policy {
	return Policy{
		context:   context,
		placement: "",
		functions: functions,
	}
}

// Gets the constraint of the policy. No two functions in the policy can have SENDER and RECEIVER constraints.
func (p *Policy) GetConstraint() ConstraintType {
	constraint := SENDER_RECEIVER
	for _, pf := range p.functions {
		if pf.GetConstraint() != SENDER_RECEIVER {
			constraint = pf.GetConstraint()
		}
	}
	return constraint
}

func (po *PolicyOptimizer) GetPlacement(policies []string) {
	// Iterate over the policies.
	for _, policy := range policies {
		// Read the json file from the jsonDir.
		b, err := os.ReadFile(path.Join(po.jsonDir, policy))
		if err != nil {
			glog.Errorf("Error reading file %s: %v", policy, err)
		}

		// Get the context from the json file.
		matchesArray, _, _, err := jsonparser.Get(b, "groups", "[0]", "inner", "Policy", "matches")
		if err != nil {
			glog.Errorf("Error getting context from file %s: %v", policy, err)
		}

		// Iterate over the objects in the matches array and get the one which has the context.
		var contextObject []byte
		jsonparser.ArrayEach(matchesArray, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			// Get the context from the object.
			object, _, _, e := jsonparser.Get(value, "Context")
			if e == nil {
				contextObject = object
			}
		})

		// Get the list of endpoints from the context object.
		endpointsArray, _, _, err := jsonparser.Get(contextObject, "blocks")
		if err != nil {
			glog.Errorf("Found context but no blocks in file %s.", policy)
		}
		glog.Infof("Context: %s", endpointsArray)
	}
}

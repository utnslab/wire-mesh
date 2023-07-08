package xPlane

type ConstraintType int

const (
	SENDER ConstraintType = iota
	RECEIVER
	SENDER_RECEIVER
)

type PolicyFunction struct {
	functionName string
	constraint   ConstraintType
	mutability   bool
}

type Policy struct {
	context   []string
	placement string
	functions []PolicyFunction
}

// Accessor methods for PolicyFunction struct.
func (pf *PolicyFunction) GetFunctionName() string {
	return pf.functionName
}

func (pf *PolicyFunction) GetConstraint() ConstraintType {
	return pf.constraint
}

func (pf *PolicyFunction) GetMutability() bool {
	return pf.mutability
}

// Create a new PolicyFunction struct.
func CreatePolicyFunction(functionName string, constraint ConstraintType, mutability bool) PolicyFunction {
	return PolicyFunction{
		functionName: functionName,
		constraint:   constraint,
		mutability:   mutability,
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

func (p *Policy) ExistsMutableFunction() bool {
	// Iterate over the functions and check if any of them mutate the CNO.
	for _, pf := range p.functions {
		if pf.GetMutability() {
			return true
		}
	}
	return false
}

// Create a new Policy struct.
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

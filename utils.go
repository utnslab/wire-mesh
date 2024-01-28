package xPlane

import (
	"golang.org/x/exp/slices"
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
	mutability   bool
	dataplanes   []int
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

func (pf *PolicyFunction) GetDataplanes() []int {
	return pf.dataplanes
}

// Create a new PolicyFunction struct.
func CreatePolicyFunction(functionName string, constraint ConstraintType, mutability bool) PolicyFunction {
	return PolicyFunction{
		functionName: functionName,
		constraint:   constraint,
		mutability:   mutability,
	}
}

// Create a new PolicyFunction struct.
func CreateNewPolicyFunction(functionName string, constraint ConstraintType, supportedDataplanes []int, mutability bool) PolicyFunction {
	return PolicyFunction{
		functionName: functionName,
		constraint:   constraint,
		mutability:   mutability,
		dataplanes:   supportedDataplanes,
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

func (p *Policy) GetDataplanes() []int {
	dataplanes := map[int]bool{}

	// Add dataplanes of the first function.
	for _, d := range p.functions[0].GetDataplanes() {
		dataplanes[d] = true
	}

	// Take intersection of dataplanes of all functions.
	for _, pf := range p.functions {
		// For every d in dataplanes, check if it exists in pf.dataplanes.
		for d := range dataplanes {
			if !slices.Contains(pf.GetDataplanes(), d) {
				delete(dataplanes, d)
			}
		}
	}

	// Whatever left, supports all functions.
	var dataplanesArray []int
	for d := range dataplanes {
		dataplanesArray = append(dataplanesArray, d)
	}

	return dataplanesArray
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

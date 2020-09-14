package session

import (
	"reflect"
	"sync"
)

type Policy interface {
	Name() string
	Initialize()
	JobCustomFieldsType() reflect.Type
	Execute(ssn *Session)
	UnInitialize()
}

var policyMutex sync.Mutex

// *Policy management
var policyMap = map[string]Policy{}

// RegisterPolicy register policy
func RegisterPolicy(policy Policy) {
	policyMutex.Lock()
	defer policyMutex.Unlock()

	policyMap[policy.Name()] = policy
}

// GetPolicy get the policy by name
func GetPolicy(name string) (Policy, bool) {
	policyMutex.Lock()
	defer policyMutex.Unlock()

	policy, found := policyMap[name]
	return policy, found
}

package fcfs

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	"reflect"
)

type JobCustomFields struct{}

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (fcfs *Policy) Name() string {
	return "fcfs"
}

func (hell *Policy) JobCustomFieldsType() reflect.Type {
	return reflect.TypeOf((*JobCustomFields)(nil))
}

func (fcfs *Policy) Initialize(in interface{}) {}

func (fcfs *Policy) Execute(snapshot *api.ClusterInfo) {
	// Do nothing
}

func (fcfs *Policy) UnInitialize() {}

package nop

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	"reflect"
)

type JobCustomFields struct {
	NumMasters  int32 `yaml:"numMasters"`
	NumReplicas int32 `yaml:"numReplicas"`
}

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (nop *Policy) Name() string {
	return "nop"
}

func (hell *Policy) JobCustomFieldsType() reflect.Type {
	return reflect.TypeOf((*JobCustomFields)(nil))
}

func (nop *Policy) Initialize() {}

func (nop *Policy) Execute(snapshot *api.ClusterInfo) {
	// Job spec passthrough
	for _, job := range snapshot.Jobs {
		customFields := job.CustomFields.(*JobCustomFields)
		job.NumMasters = customFields.NumMasters
		job.NumReplicas = customFields.NumReplicas
	}
}

func (nop *Policy) UnInitialize() {}

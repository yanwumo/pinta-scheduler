package hell

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	"gopkg.in/yaml.v2"
	"reflect"
)

type JobCustomFields struct {
	BatchSize   int       `yaml:"batchSize"`
	Iterations  int       `yaml:"iterations"`
	Performance []float64 `yaml:"performance"`
}

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (hell *Policy) Name() string {
	return "hell"
}

func (hell *Policy) JobCustomFieldsType() reflect.Type {
	return reflect.TypeOf((*JobCustomFields)(nil))
}

func (hell *Policy) Initialize() {}

func (hell *Policy) Execute(snapshot *api.ClusterInfo) {
	// Do nothing
}

func (hell *Policy) UnInitialize() {}

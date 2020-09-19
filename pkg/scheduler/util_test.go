package scheduler

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/conf"
	_ "github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/nop"
	"reflect"
	"testing"
)

func TestLoadSchedulerConf(t *testing.T) {
	schedulerConf := `
policy: "nop"
configuration:
  arguments:
    k: 1.2
    b: true
`
	expectedPolicy := &nop.Policy{}
	expectedConfiguration := &conf.Configuration{
		Arguments: map[string]string{
			"k": "1.2",
			"b": "true",
		},
	}

	policy, configuration, err := loadSchedulerConf(schedulerConf)
	if err != nil {
		t.Errorf("Failed to load scheduler configuration: %v", err)
	}
	if !reflect.DeepEqual(policy, expectedPolicy) {
		t.Errorf("Failed to set default settings for policies, expected: %+v, got %+v",
			expectedPolicy, policy)
	}
	if !reflect.DeepEqual(configuration, expectedConfiguration) {
		t.Errorf("Wrong configuration, expected: %+v, got %+v",
			expectedConfiguration, configuration)
	}
}

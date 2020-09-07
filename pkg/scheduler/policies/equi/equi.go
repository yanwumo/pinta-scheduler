package equi

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	"reflect"
)

type JobCustomFields struct{}

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (equi *Policy) Name() string {
	return "equi"
}

func (hell *Policy) JobCustomFieldsType() reflect.Type {
	return reflect.TypeOf((*JobCustomFields)(nil))
}

func (equi *Policy) Initialize(in interface{}) {}

func (equi *Policy) Execute(snapshot *api.ClusterInfo) {
	numNodes := len(snapshot.Nodes)

	if len(snapshot.Jobs) == 0 {
		return
	}
	// 1st judge
	judge := make(map[int32]bool)

	for _, job := range snapshot.Jobs {
		judge[job.NumReplicas] = true
	}
	if len(judge) <= 2 {
		// 2nd judge
		sumReplicas := 0
		for _, job := range snapshot.Jobs {
			sumReplicas += int(job.NumReplicas)
		}
		if numNodes == sumReplicas {
			// 3rd judge
			if len(judge) == 1 {
				return
			}
			var judgeArr []int
			for key := range judge {
				judgeArr = append(judgeArr, int(key))
			}
			if judgeArr[0]-judgeArr[1] == 1 || judgeArr[0]-judgeArr[1] == -1 {
				return
			}
		}
	}

	for _, job := range snapshot.Jobs {
		job.NumReplicas = 0
	}
	for {
		for _, job := range snapshot.Jobs {
			job.NumReplicas++
			numNodes--
			if numNodes == 0 {
				return
			}
		}
	}
}

func (equi *Policy) UnInitialize() {}

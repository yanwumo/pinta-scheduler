package equi

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/session"
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

func (equi *Policy) JobCustomFieldsType() reflect.Type {
	return reflect.TypeOf((*JobCustomFields)(nil))
}

func (equi *Policy) Initialize() {}

func (equi *Policy) Execute(ssn *session.Session) {
	numNodes := len(ssn.Nodes)

	if len(ssn.Jobs) == 0 {
		return
	}
	// 1st judge
	judge := make(map[int32]bool)

	for _, job := range ssn.Jobs {
		judge[job.NumReplicas] = true
	}
	if len(judge) <= 2 {
		// 2nd judge
		sumReplicas := 0
		for _, job := range ssn.Jobs {
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

	for _, job := range ssn.Jobs {
		job.NumReplicas = 0
	}
	for {
		for _, job := range ssn.Jobs {
			job.NumReplicas++
			numNodes--
			if numNodes == 0 {
				return
			}
		}
	}
}

func (equi *Policy) UnInitialize() {}

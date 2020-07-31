package policies

import "github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/fcfs"

func init() {
	RegisterPolicy(fcfs.New())
}

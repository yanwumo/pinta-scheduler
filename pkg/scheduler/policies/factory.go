package policies

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/equi"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/fcfs"
)

func init() {
	RegisterPolicy(equi.New())
	RegisterPolicy(fcfs.New())
}

package policies

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/equi"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/fcfs"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/hell"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/nop"
)

func init() {
	RegisterPolicy(nop.New())
	RegisterPolicy(fcfs.New())
	RegisterPolicy(equi.New())
	RegisterPolicy(hell.New())
}

package policies

import (
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/equi"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/fcfs"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/hell"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/policies/nop"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/session"
)

func init() {
	session.RegisterPolicy(nop.New())
	session.RegisterPolicy(fcfs.New())
	session.RegisterPolicy(equi.New())
	session.RegisterPolicy(hell.New())
}

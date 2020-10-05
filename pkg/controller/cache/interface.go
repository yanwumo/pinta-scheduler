package cache

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/controller/api"
	"k8s.io/api/core/v1"
)

type Cache interface {
	Run(stopCh <-chan struct{})

	Get(key string) (*api.JobInfo, error)
	GetStatus(key string) (*pintav1.PintaJobStatus, error)
	Add(obj *pintav1.PintaJob) error
	Update(obj *pintav1.PintaJob) error
	Delete(obj *pintav1.PintaJob) error

	AddNode(node *v1.Node)
	UpdateNode(node *v1.Node)
	DeleteNode(node *v1.Node)
}

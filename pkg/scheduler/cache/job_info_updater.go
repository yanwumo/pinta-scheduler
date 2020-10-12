package cache

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	clientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	v1 "k8s.io/api/core/v1"
)

type defaultJobInfoUpdater struct {
	pintaClient *clientset.Clientset
}

func (jiu *defaultJobInfoUpdater) UpdateJobNodeType(nodeType string) error {
	return nil
}

func (jiu *defaultJobInfoUpdater) UpdateJobResourceRequirements(rl v1.ResourceList) error {
	panic("implement me")
}

func (jiu *defaultJobInfoUpdater) UpdateJobStatus(status pintav1.PintaJobStatus) error {
	panic("implement me")
}

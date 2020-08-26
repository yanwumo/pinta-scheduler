package api

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
)

type JobID types.UID

type JobInfo struct {
	UID       JobID
	Name      string
	Namespace string

	NumMasters  int32
	NumReplicas int32

	CreationTimestamp metav1.Time

	CustomFields interface{}

	Job *pintav1.PintaJob
}

func NewJobInfo(uid JobID, job *pintav1.PintaJob) *JobInfo {
	jobInfo := &JobInfo{
		UID:       uid,
		Name:      job.Name,
		Namespace: job.Namespace,

		NumMasters:  job.Status.NumMasters,
		NumReplicas: job.Status.NumReplicas,

		CreationTimestamp: job.GetCreationTimestamp(),

		Job: job,
	}
	return jobInfo
}

func (ji *JobInfo) ParseCustomFields(customFieldsType reflect.Type) {
	customFieldsInterface := reflect.New(customFieldsType.Elem()).Interface()
	customFieldsStr := ji.Job.GetAnnotations()["pinta.qed.usc.edu/custom-fields"]
	err := yaml.Unmarshal([]byte(customFieldsStr), customFieldsInterface)
	if err == nil {
		ji.CustomFields = customFieldsInterface
	}
}

func (ji *JobInfo) Clone() *JobInfo {
	info := &JobInfo{
		UID:          ji.UID,
		Name:         ji.Name,
		Namespace:    ji.Namespace,
		NumMasters:   ji.NumMasters,
		NumReplicas:  ji.NumReplicas,
		CustomFields: ji.CustomFields,
		Job:          ji.Job.DeepCopy(),
	}

	ji.CreationTimestamp.DeepCopyInto(&info.CreationTimestamp)

	return info
}

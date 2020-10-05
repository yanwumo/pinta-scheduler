package info

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"testing"
	"time"
)

func buildPintaJob(name string, timestamp metav1.Time) *pintav1.PintaJob {
	return &pintav1.PintaJob{
		ObjectMeta: metav1.ObjectMeta{
			UID:               types.UID(name),
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: timestamp,
		},
		Spec: pintav1.PintaJobSpec{
			Type: "type1",
		},
		Status: []pintav1.PintaJobStatus{
			{
				NumMasters:  1,
				NumReplicas: 2,
			},
		},
	}
}

func buildPintaJobWithCustomFields(name string, timestamp metav1.Time, customFieldsStr string) *pintav1.PintaJob {
	return &pintav1.PintaJob{
		ObjectMeta: metav1.ObjectMeta{
			UID:               types.UID(name),
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: timestamp,
			Annotations: map[string]string{
				"pinta.qed.usc.edu/custom-fields": customFieldsStr,
			},
		},
		Spec: pintav1.PintaJobSpec{
			Type: "type1",
		},
		Status: []pintav1.PintaJobStatus{
			{
				NumMasters:  1,
				NumReplicas: 2,
			},
		},
	}
}

func TestNewJobInfo(t *testing.T) {
	ts := metav1.NewTime(time.Now())
	t1job := buildPintaJob("j1", ts)

	tests := []struct {
		id       JobID
		job      *pintav1.PintaJob
		expected *JobInfo
	}{
		{
			id:  JobID("j1"),
			job: t1job,
			expected: &JobInfo{
				UID:               JobID("j1"),
				Name:              "j1",
				Namespace:         "default",
				Type:              "type1",
				NumMasters:        1,
				NumReplicas:       2,
				CreationTimestamp: ts,
				CustomFields:      nil,
				Job:               t1job,
			},
		},
	}

	for i, test := range tests {
		ji := NewJobInfo(test.id, test.job)

		if !reflect.DeepEqual(ji, test.expected) {
			t.Errorf("job info %d: \n expected %v, \n got %v \n",
				i, test.expected, ji)
		}
	}
}

func TestJobInfo_ParseCustomFields(t *testing.T) {
	type CustomFields struct {
		I   int     `yaml:"i"`
		F   float64 `yaml:"f"`
		S   string  `yaml:"s"`
		Opt string  `yaml:"opt,omitempty"`
	}
	customFieldsStr := `
i: 1
f: 1.2
s: test
`
	ts := metav1.NewTime(time.Now())
	t1job := buildPintaJobWithCustomFields("j1", ts, customFieldsStr)
	customFields := CustomFields{
		I: 1,
		F: 1.2,
		S: "test",
	}

	tests := []struct {
		id       JobID
		job      *pintav1.PintaJob
		expected *JobInfo
	}{
		{
			id:  JobID("j1"),
			job: t1job,
			expected: &JobInfo{
				UID:               JobID("j1"),
				Name:              "j1",
				Namespace:         "default",
				Type:              "type1",
				NumMasters:        1,
				NumReplicas:       2,
				CreationTimestamp: ts,
				CustomFields:      &customFields,
				Job:               t1job,
			},
		},
	}

	for i, test := range tests {
		ji := NewJobInfo(test.id, test.job)
		err := ji.ParseCustomFields(reflect.TypeOf((*CustomFields)(nil)))
		if err != nil {
			t.Errorf("Cannot parse custom fields for job %v: %v", ji.Name, err)
		}

		if !reflect.DeepEqual(ji, test.expected) {
			t.Errorf("job info %d: \n expected %v, \n got %v \n",
				i, test.expected, ji)
		}
	}
}

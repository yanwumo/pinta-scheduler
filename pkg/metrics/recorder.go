package metrics

import (
	"fmt"
	"k8s.io/klog"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	v1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
)

type Recorder struct {
	metrics map[string]interface{}
}

func NewRecorder() *Recorder {
	return &Recorder{
		metrics: make(map[string]interface{}),
	}
}

func (r *Recorder) PintaJobStatusMetric(pintaJob *v1.PintaJob, fromStatus, toStatus v1.PintaJobState) {
	switch toStatus {
	case v1.Idle:
		r.pintaJobStatusStart(pintaJob.GetNamespace(), pintaJob.GetName(), v1.Idle, totalPintaJobs)
	case v1.Completed:
		r.pintaJobStatusEnd(pintaJob.GetNamespace(), pintaJob.GetName(), v1.Idle, pintaJobTime, succeededPintaJobs)
	case v1.Failed:
	case v1.Preempted:
	case v1.Running:
	case v1.Scheduled:
	default:
	}

}

// pintaJobStatusStart starts to observe a metric.
func (r *Recorder) pintaJobStatusStart(ns, jobName string, status v1.PintaJobState, counterInc prometheus.Counter) {
	r.entryStart(jobIdentity(ns, jobName, status), counterInc)
}

// pintaJobStatusSample collects the time duration of pintajob from a status it started in.
func (r *Recorder) pintaJobStatusSample(ns, jobName string, fromStatus v1.PintaJobState, summary prometheus.Summary, counterInc prometheus.Counter) {
	entry := jobIdentity(ns, jobName, fromStatus)
	durSinceIdle, err := r.entryEnd(entry)
	if err != nil {
		klog.V(4).Infof("error recording entry: %s, %v", entry, err)
	} else {
		summary.Observe(durSinceIdle.Seconds())
		counterInc.Inc()
	}
}

// pintaJobStatusEnd observe the metric like pintaJobStatusSample and deletes from map.
func (r *Recorder) pintaJobStatusEnd(ns, jobName string, fromStatus v1.PintaJobState, summary prometheus.Summary, counterInc prometheus.Counter) {
	r.pintaJobStatusSample(ns, jobName, fromStatus, summary, counterInc)
	delete(r.metrics, jobIdentity(ns, jobName, fromStatus))
}

func (r *Recorder) entryStart(entry string, counterInc prometheus.Counter) {
	if _, ok := r.metrics[entry]; !ok {
		r.metrics[entry] = time.Now()
		counterInc.Inc()
	}
}

func (r *Recorder) entryEnd(entry string) (time.Duration, error) {
	start, ok := r.metrics[entry]
	if !ok {
		return 0, fmt.Errorf("entry not exist to provide a duration")
	}
	startTime, ok := start.(time.Time)
	if !ok {
		return 0, fmt.Errorf("entry does not have a time entry: %v", start)
	}
	return time.Now().Sub(startTime), nil
}

func jobIdentity(ns, name string, status v1.PintaJobState) string {
	return strings.Join([]string{ns, name, string(status)}, ":")
}

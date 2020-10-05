package hell

import (
	"bytes"
	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/session"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
	"math"
	"reflect"
	"strconv"
)

type JobCustomFields struct {
	BatchSize  int       `yaml:"batchSize"`
	Iterations int       `yaml:"iterations"`
	Throughput []float64 `yaml:"throughput"`

	CompletedIterations int
}

type Policy struct{}

func New() *Policy {
	return &Policy{}
}

func (hell *Policy) Name() string {
	return "hell"
}

func (hell *Policy) JobCustomFieldsType() reflect.Type {
	return reflect.TypeOf((*JobCustomFields)(nil))
}

func (hell *Policy) Initialize() {}

func (hell *Policy) Execute(ssn *session.Session) {
	klog.V(3).Infof("Begin HELL")
	defer klog.V(3).Infof("End HELL")

	// Get # completed iterations reported by the job
	for _, job := range ssn.Jobs {
		var podName string
		switch job.Type {
		case pintav1.Symmetric:
			podName = job.Name + "-replica-0"
		case pintav1.PSWorker:
			podName = job.Name + "-worker-0"
		case pintav1.MPI:
			podName = job.Name + "-replica-0"
		default:
			continue
		}
		clientset := ssn.KubeClient()
		req := clientset.CoreV1().RESTClient().Post().Resource("pods").
			Name(podName).
			Namespace(job.Namespace).SubResource("exec")
		req.VersionedParams(&v1.PodExecOptions{
			Command: []string{"cat", "/etc/pinta/ITERATION"},
			Stdin:   false,
			Stdout:  true,
			Stderr:  false,
			TTY:     false,
		}, scheme.ParameterCodec)

		var stdout bytes.Buffer
		exec, err := remotecommand.NewSPDYExecutor(ssn.KubeConfig(), "POST", req.URL())
		if err != nil {
			continue
		}
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  nil,
			Stdout: &stdout,
			Stderr: nil,
		})
		customFields := job.CustomFields.(*JobCustomFields)
		stdoutString := stdout.String()
		customFields.CompletedIterations, err = strconv.Atoi(stdoutString)
		if err != nil {
			continue
		}
	}

	// Clear previous schedules
	for _, job := range ssn.Jobs {
		job.NumMasters = 0
		job.NumReplicas = 0
	}

	// Calculate ratios
	ratiosMap := make(map[info.JobID][]float64)                // Schedule based on ratios
	remainingServiceTimesMap := make(map[info.JobID][]float64) // Fill based on remaining service times
	for id, job := range ssn.Jobs {
		customFields := job.CustomFields.(*JobCustomFields)
		throughput := customFields.Throughput
		remainingIterations := customFields.Iterations - customFields.CompletedIterations
		remainingExamples := remainingIterations * customFields.BatchSize
		remainingServiceTimes := make([]float64, len(throughput))
		remainingServiceTimesMap[id] = remainingServiceTimes
		ratios := make([]float64, len(throughput))
		ratiosMap[id] = ratios
		for i := range throughput {
			speedup := throughput[i] / throughput[0]
			efficiency := speedup / float64(i+1)
			remainingServiceTimes[i] = float64(remainingExamples) / throughput[i]
			ratios[i] = remainingServiceTimes[i] / efficiency
		}
	}

	// Schedule
	numNodes := len(ssn.Nodes)
	for numNodes > 0 && len(ratiosMap) > 0 {
		// Pick the job with minimum ratio
		var nextJob *info.JobInfo
		optimalNumReplicas := 0
		minRatio := math.MaxFloat64
		for id, ratios := range ratiosMap {
			job := ssn.Jobs[id]
			numMasters := 0
			if job.Type == pintav1.PSWorker || job.Type == pintav1.MPI {
				numMasters = 1
			}
			for i := 0; i < numNodes-numMasters && i < len(ratios); i++ {
				change := false
				if ratios[i] < minRatio {
					change = true
				} else if ratios[i] == minRatio {
					if nextJob == nil {
						continue
					}
					// Break ties
					oldTime := nextJob.CreationTimestamp
					if job.CreationTimestamp.Before(&oldTime) {
						change = true
					} else if job.CreationTimestamp.Equal(&oldTime) {
						if job.UID < nextJob.UID {
							change = true
						}
					}
				}
				if change {
					nextJob = job
					optimalNumReplicas = i + 1
					minRatio = ratios[i]
				}
			}
		}
		if nextJob == nil {
			klog.Errorf("No PintaJob to schedule: %d node available", numNodes)
			break
		}
		// Schedule
		if nextJob.Type == pintav1.PSWorker || nextJob.Type == pintav1.MPI {
			nextJob.NumMasters = 1
		}
		nextJob.NumReplicas = int32(optimalNumReplicas)
		numNodes -= int(nextJob.NumMasters + nextJob.NumReplicas)
		delete(ratiosMap, nextJob.UID)
	}
	ratiosMap = nil

	// Fill
	for numNodes > 0 && len(remainingServiceTimesMap) > 0 {
		// Pick the job with min # replicas to achieve min remaining service time
		minAdditionalNumReplicasToAchieveMinRemainingServiceTime := math.MaxInt32
		var nextJob *info.JobInfo
		for id, remainingServiceTimes := range remainingServiceTimesMap {
			job := ssn.Jobs[id]
			if job.NumReplicas == 0 {
				delete(remainingServiceTimesMap, job.UID)
				continue
			}
			var numAdditionalReplicasToAchieveMinRemainingServiceTime int
			minRemainingServiceTime := math.MaxFloat64
			for additionalNodes := 0; additionalNodes <= numNodes; additionalNodes++ {
				if int(job.NumReplicas)+additionalNodes > len(remainingServiceTimes) {
					break
				}
				if remainingServiceTimes[int(job.NumReplicas)+additionalNodes-1] < minRemainingServiceTime {
					minRemainingServiceTime = remainingServiceTimes[int(job.NumReplicas)+additionalNodes-1]
					numAdditionalReplicasToAchieveMinRemainingServiceTime = additionalNodes
				}
			}
			change := false
			if numAdditionalReplicasToAchieveMinRemainingServiceTime < minAdditionalNumReplicasToAchieveMinRemainingServiceTime {
				change = true
			} else if numAdditionalReplicasToAchieveMinRemainingServiceTime == minAdditionalNumReplicasToAchieveMinRemainingServiceTime {
				if nextJob == nil {
					continue
				}
				// Break ties
				oldTime := nextJob.CreationTimestamp
				if job.CreationTimestamp.Before(&oldTime) {
					change = true
				} else if job.CreationTimestamp.Equal(&oldTime) {
					if job.UID < nextJob.UID {
						change = true
					}
				}
			}
			if change {
				minAdditionalNumReplicasToAchieveMinRemainingServiceTime = numAdditionalReplicasToAchieveMinRemainingServiceTime
				nextJob = job
			}
		}

		if nextJob == nil {
			break
		}

		nextJob.NumReplicas += int32(minAdditionalNumReplicasToAchieveMinRemainingServiceTime)
		numNodes -= minAdditionalNumReplicasToAchieveMinRemainingServiceTime
		delete(remainingServiceTimesMap, nextJob.UID)
	}
	remainingServiceTimesMap = nil
}

func (hell *Policy) UnInitialize() {}

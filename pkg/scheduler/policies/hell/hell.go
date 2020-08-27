package hell

import (
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	"k8s.io/klog"
	"math"
	"reflect"
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

func (hell *Policy) Execute(snapshot *api.ClusterInfo) {
	klog.V(3).Infof("Begin HELL")
	defer klog.V(3).Infof("End HELL")

	// Clear previous schedules
	for _, job := range snapshot.Jobs {
		job.NumMasters = 0
		job.NumReplicas = 0
	}

	// Calculate ratios
	ratiosMap := make(map[api.JobID][]float64)                // Schedule based on ratios
	remainingServiceTimesMap := make(map[api.JobID][]float64) // Fill based on remaining service times
	for id, job := range snapshot.Jobs {
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
	numNodes := len(snapshot.Nodes)
	for numNodes > 0 && len(ratiosMap) > 0 {
		// Pick the job with minimum ratio
		var nextJob *api.JobInfo
		optimalNumReplicas := 0
		minRatio := math.MaxFloat64
		for id, ratios := range ratiosMap {
			job := snapshot.Jobs[id]
			numMasters := 0
			if job.Type == pintav1.PSWorker || job.Type == pintav1.MPI {
				numMasters = 1
			}
			for i := 0; i < numNodes-numMasters && i < len(ratios); i++ {
				if ratios[i] < minRatio {
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
		minAdditionalNumReplicasToAchieveMinRemainingServiceTime := math.MaxInt32
		var nextJob *api.JobInfo
		for id, remainingServiceTimes := range remainingServiceTimesMap {
			job := snapshot.Jobs[id]
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

			if numAdditionalReplicasToAchieveMinRemainingServiceTime < minAdditionalNumReplicasToAchieveMinRemainingServiceTime {
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

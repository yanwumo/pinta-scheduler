package metrics

import "github.com/prometheus/client_golang/prometheus"

const subsysPintaJob string = "pinta_job"

var (
	pintaJobQueueTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: subsysPintaJob,
		Name:      "queueing_seconds",
		Help:      "Pinta job queueing time before scheduler make a decision.",
	})

	pintaJobScheduleTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: subsysPintaJob,
		Name:      "scheduling_seconds",
		Help:      "Pinta job scheduling time which is the time that scheduler uses to filter and prioritize nodes for job pods.",
	})

	pintaJobPendingTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: subsysPintaJob,
		Name:      "pending_seconds",
		Help:      "Pinta job overheads from scheduled to available state, which includes image pulling and pod initiating.",
	})

	pintaJobServiceTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: subsysPintaJob,
		Name:      "service_seconds",
		Help:      "Pinta job running time from available to done.",
	})

	pintaJobTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: subsysPintaJob,
		Name:      "time_seconds",
		Help:      "Pinta job time from entering the system to done.",
	})

	totalPintaJobs = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: subsysPintaJob,
		Name:      "total_count",
		Help:      "Total number of pinta jobs created.",
	})

	// PintaJobs still in queue = totalPintaJobs - scheduledPintaJobs
	scheduledPintaJobs = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: subsysPintaJob,
		Name:      "scheduled_count",
		Help:      "Number of successfully scheduled pinta jobs.",
	})

	// failedPintaJobs = scheduledPintaJobs - succeededPintaJobs
	succeededPintaJobs = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: subsysPintaJob,
		Name:      "succeeded_count",
		Help:      "Number of succeeded pinta jobs.",
	})

	preemptedPintaJobs = prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: subsysPintaJob,
		Name:      "preempted_count",
		Help:      "Number of preempted pinta jobs.",
	})
)

func RegisterPintaJob() {
	prometheus.MustRegister(pintaJobQueueTime, pintaJobScheduleTime, pintaJobPendingTime, pintaJobServiceTime, pintaJobTime)
	prometheus.MustRegister(totalPintaJobs, scheduledPintaJobs, succeededPintaJobs, preemptedPintaJobs)
}

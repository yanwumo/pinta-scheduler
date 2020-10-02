package api

import v1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"

type GeneratorSpec struct {
	Templates []Template `json:"templates,omitempty"`
	Jobs      JobSpec    `json:"jobs,omitempty"`
}

type Template struct {
	Job    v1.PintaJob `json:"job,omitempty"`
	Weight float64     `json:"weight,omitempty"`
}

type JobSpec struct {
	Poisson PoissonJobsSpec `json:"poisson,omitempty"`
	//Times []int   `json:"times,omitempty"`
}

type PoissonJobsSpec struct {
	Rate    float64 `json:"rate,omitempty"`
	NumJobs int     `json:"numJobs,omitempty"`
}

type SimulatorSpec struct {
	Jobs []SimulationJob `json:"jobs,omitempty"`
}

type SimulationJob struct {
	Time float64     `json:"time"`
	Job  v1.PintaJob `json:"job,omitempty"`
}

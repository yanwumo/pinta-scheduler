package generator

import (
	"github.com/qed-usc/pinta-scheduler/cmd/generator/options"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	simulationapi "github.com/qed-usc/pinta-scheduler/pkg/apis/simulation"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"math/rand"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
	"time"
)

type Generator struct {
	spec    *simulationapi.GeneratorSpec
	fileOut string
}

func NewGenerator(opt *options.Option) (*Generator, error) {
	content, err := ioutil.ReadFile(opt.FileIn)
	if err != nil {
		return nil, err
	}

	var spec simulationapi.GeneratorSpec
	err = yaml.Unmarshal(content, &spec)

	generator := &Generator{
		spec:    &spec,
		fileOut: opt.FileOut,
	}
	if err != nil {
		return nil, err
	}

	return generator, nil
}

func (g *Generator) Run() error {
	rand.Seed(time.Now().Unix())
	return g.GenerateJobs()
}

func (g *Generator) GenerateJobs() error {
	var spec simulationapi.SimulatorSpec
	timestamp := 0.0
	id := 0
	for i := 0; i < g.spec.Jobs.Poisson.NumJobs; i++ {
		// Choose a random job spec from templates
		template := g.spec.Templates[rand.Intn(len(g.spec.Templates))]

		job, err := CreateJobFromTemplate(&template.Job, id)
		if err != nil {
			return err
		}

		spec.Jobs = append(spec.Jobs, simulationapi.SimulationJob{
			Time: timestamp,
			Job:  *job,
		})

		rate := g.spec.Jobs.Poisson.Rate
		interval := rand.ExpFloat64() / rate
		timestamp += interval
		id++
	}

	specOut, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(g.fileOut, specOut, 0644)
	if err != nil {
		return err
	}

	return nil
}

func CreateJobFromTemplate(template *pintav1.PintaJob, id int) (*pintav1.PintaJob, error) {
	job := template.DeepCopy()

	// Inject job
	job.Name = "pinta-job-simulation-" + strconv.Itoa(id)

	// Remove existing envs
	env := make([]v1.EnvVar, 0, len(job.Spec.Replica.Spec.Containers[0].Env))
	for _, e := range job.Spec.Replica.Spec.Containers[0].Env {
		if e.Name == "BATCH_SIZE" || e.Name == "ITERATIONS" || e.Name == "THROUGHPUT" {
			continue
		}
		env = append(env, e)
	}

	// Insert envs
	var customFields struct {
		BatchSize  string   `yaml:"batchSize"`
		Iterations string   `yaml:"iterations"`
		Throughput []string `yaml:"throughput"`
	}
	customFieldsStr := job.GetAnnotations()["pinta.qed.usc.edu/custom-fields"]
	err := yaml.Unmarshal([]byte(customFieldsStr), &customFields)
	if err != nil {
		return nil, err
	}
	env = append(env, v1.EnvVar{
		Name:  "BATCH_SIZE",
		Value: customFields.BatchSize,
	}, v1.EnvVar{
		Name:  "ITERATIONS",
		Value: customFields.Iterations,
	}, v1.EnvVar{
		Name:  "THROUGHPUT",
		Value: strings.Join(customFields.Throughput, ","),
	})
	job.Spec.Replica.Spec.Containers[0].Env = env

	return job, nil
}

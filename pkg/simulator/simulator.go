package simulator

import (
	"context"
	"fmt"
	"github.com/qed-usc/pinta-scheduler/cmd/simulator/options"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	simulationapi "github.com/qed-usc/pinta-scheduler/pkg/apis/simulation"
	pintaclientset "github.com/qed-usc/pinta-scheduler/pkg/generated/clientset/versioned"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"math/rand"
	"sigs.k8s.io/yaml"
	"time"
)

type Simulator struct {
	kubeConfig    *rest.Config
	pintaClient   *pintaclientset.Clientset
	spec          *simulationapi.SimulatorSpec
	baseTimestamp time.Time
}

func NewSimulator(config *rest.Config, opt *options.Option) (*Simulator, error) {
	pintaClinet, err := pintaclientset.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("Kubernetes clientset initialization failed: %v", err))
	}

	content, err := ioutil.ReadFile(opt.FileIn)
	if err != nil {
		return nil, err
	}

	var spec simulationapi.SimulatorSpec
	err = yaml.Unmarshal(content, &spec)

	simulator := &Simulator{
		kubeConfig:    config,
		pintaClient:   pintaClinet,
		spec:          &spec,
		baseTimestamp: time.Now(),
	}
	if err != nil {
		return nil, err
	}

	simulator.log("Simulator created")
	return simulator, nil
}

func (s *Simulator) Run() error {
	rand.Seed(time.Now().Unix())
	for _, jobSpec := range s.spec.Jobs {
		currentRelativeTime := time.Now().Sub(s.baseTimestamp).Seconds()
		jobRelativeTime := jobSpec.Time
		if currentRelativeTime < jobRelativeTime {
			remainingTime := int64((jobRelativeTime - currentRelativeTime) * 1e9)
			timer := time.NewTimer(time.Duration(remainingTime))

			<-timer.C
		}

		err := s.CreateJob(&jobSpec.Job)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Simulator) CreateJob(job *pintav1.PintaJob) error {
	_, err := s.pintaClient.PintaV1().PintaJobs("default").Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	s.log("Job created")
	return nil
}

func (s *Simulator) log(str string) {
	diff := time.Now().Sub(s.baseTimestamp)
	fmt.Printf("T+%v: %v\n", diff, str)
}

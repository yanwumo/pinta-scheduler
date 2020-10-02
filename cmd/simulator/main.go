package main

import (
	"fmt"
	"github.com/qed-usc/pinta-scheduler/cmd/simulator/options"
	"github.com/qed-usc/pinta-scheduler/pkg/kube"
	"github.com/qed-usc/pinta-scheduler/pkg/simulator"
	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
	"os"
)

func main() {
	o := options.NewOption()
	o.AddFlags(pflag.CommandLine)

	cliflag.InitFlags()
	if err := o.CheckOptionOrDie(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := Run(o); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Run(opt *options.Option) error {
	config, err := kube.BuildConfig(opt.KubeClientOptions)
	if err != nil {
		return err
	}

	sim, err := simulator.NewSimulator(config, opt)
	if err != nil {
		return err
	}

	err = sim.Run()
	if err != nil {
		return err
	}

	return nil
}

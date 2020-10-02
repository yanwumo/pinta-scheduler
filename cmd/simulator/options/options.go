package options

import (
	"fmt"
	"github.com/qed-usc/pinta-scheduler/pkg/kube"
	"github.com/spf13/pflag"
)

type Option struct {
	KubeClientOptions kube.ClientOptions
	FileIn            string
}

func NewOption() *Option {
	o := Option{}
	return &o
}

func (o *Option) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeClientOptions.Master, "master", o.KubeClientOptions.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&o.KubeClientOptions.KubeConfig, "kubeconfig", o.KubeClientOptions.KubeConfig, "Path to kubeconfig file with authorization and master location information")
	fs.StringVar(&o.FileIn, "file", "", "Path to jobs specification file")
}

func (o *Option) CheckOptionOrDie() error {
	if o.FileIn == "" {
		return fmt.Errorf("jobs file must be specified")
	}
	return nil
}

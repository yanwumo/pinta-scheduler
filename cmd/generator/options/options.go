package options

import (
	"fmt"
	"github.com/spf13/pflag"
)

type Option struct {
	FileIn  string
	FileOut string
}

func NewOption() *Option {
	o := Option{}
	return &o
}

func (o *Option) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.FileIn, "in", "", "Input file")
	fs.StringVar(&o.FileOut, "out", "", "Output file")
}

func (o *Option) CheckOptionOrDie() error {
	if o.FileIn == "" {
		return fmt.Errorf("input file must be specified")
	}
	if o.FileOut == "" {
		return fmt.Errorf("output file must be specified")
	}
	return nil
}

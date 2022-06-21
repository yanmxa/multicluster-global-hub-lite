package options

import (
	"errors"

	"github.com/spf13/pflag"
)

type Options struct {
	FromKubeconfig string
	ToKubeconfig   string
}

func NewOptions() *Options {
	return &Options{}
}

func (options *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&options.FromKubeconfig, "from-kubeconfig", options.FromKubeconfig, "Kubeconfig file for -from cluster.")
}

func (options *Options) Complete() error {
	return nil
}

func (options *Options) Validate() error {
	if options.FromKubeconfig == "" {
		return errors.New("--from-kubeconfig is required")
	}

	return nil
}
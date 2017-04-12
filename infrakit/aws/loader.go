package aws

import (
	"github.com/docker/docker-e2e/infrakit"
	aws_instance "github.com/docker/infrakit.aws/plugin/instance"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/spf13/pflag"
)

// Options contain
type Options struct {
	Namespace string
}

func init() {
	infrakit.Register("aws/ec2-instance", LoadInstancePlugin)
}

var (
	builder = &aws_instance.Builder{}
	options = &Options{}
)

func Flags() *pflag.FlagSet {
	fs := builder.Flags()
	fs.StringVar(&options.Namespace, "namespace", "testkit", "Scoping label for resources")
	return fs
}

// LoadInstancePlugin creates the instance plugin
func LoadInstancePlugin() (instance.Plugin, error) {

	namespace := map[string]string{
		"testkit.scope": options.Namespace,
	}
	return builder.BuildInstancePlugin(namespace)
}

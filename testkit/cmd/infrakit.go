package cmd

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker-e2e/infrakit"
	"github.com/docker/docker-e2e/infrakit/aws"

	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	"github.com/spf13/cobra"
)

func infrakitCmd() *cobra.Command {
	provisionTemplateFlags, toJSON, _, provisionProcessTemplate := infrakit.TemplateProcessor()
	infrakit := &cobra.Command{
		Use:   "infrakit <provider> <url>",
		Short: "Create instances using infrakit",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) != 2 {
				cmd.Usage()
				return fmt.Errorf("args: provider configURL")
			}

			provider := args[0]
			templateURL := args[1]

			instancePlugin, err := infrakit.GetInstancePlugin(provider)
			if err != nil {
				return err
			}
			log.Info("infrakit found plugin", "provider", provider, "url", templateURL, "instance", instancePlugin)

			view, err := infrakit.ReadFromStdinIfElse(
				func() bool { return templateURL == "-" },
				func() (string, error) { return provisionProcessTemplate(templateURL) },
				toJSON,
			)
			if err != nil {
				return err
			}

			spec := instance.Spec{}
			if err := types.AnyString(view).Decode(&spec); err != nil {
				return err
			}

			id, err := instancePlugin.Provision(spec)
			if err == nil && id != nil {
				fmt.Printf("%s\n", *id)
			}
			return err
		},
	}
	infrakit.Flags().AddFlagSet(aws.Flags())
	infrakit.Flags().AddFlagSet(provisionTemplateFlags)

	return infrakit
}

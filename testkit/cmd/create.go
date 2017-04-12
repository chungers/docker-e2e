package cmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker-e2e/infrakit"
	"github.com/docker/docker-e2e/infrakit/aws"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	"github.com/spf13/cobra"

	"github.com/docker/docker-e2e/testkit/machines"
)

func createCmd() *cobra.Command {

	createCmd := &cobra.Command{
		Use:   "create <linux_count> <windows_count>",
		Short: "Provision a test environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return err
			}
			if debug {
				log.SetLevel(log.DebugLevel)
			}
			if len(args) == 0 {
				return errors.New("Config missing")
			}

			linuxCount, err := strconv.Atoi(args[0])
			if err != nil {
				log.Fatal(err)
			}
			windowsCount, err := strconv.Atoi(args[1])
			if err != nil {
				log.Fatal(err)
			}

			lm, wm, err := machines.GetTestMachines(linuxCount, windowsCount)
			if err != nil {
				log.Fatalf("Failure: %s", err)
			}
			noInit, err := cmd.Flags().GetBool("no-swarm")
			if err != nil {
				return err
			}
			advertiseAddr, _ := cmd.Flags().GetString("advertise-addr")
			listenAddr, _ := cmd.Flags().GetString("listen-addr")
			if !noInit {
				// Init and join
				cli, err := lm[0].GetEngineAPI()
				if err != nil {
					return err
				}
				log.Debug("Initializing swarm on %s", lm[0].GetName())
				_, err = cli.SwarmInit(context.TODO(), swarm.InitRequest{
					ListenAddr:    listenAddr,
					AdvertiseAddr: advertiseAddr,
				})
				if err != nil {
					return err
				}
				swarmInfo, err := cli.SwarmInspect(context.TODO())
				if err != nil {
					return err
				}
				info, err := cli.Info(context.TODO())
				if err != nil {
					return err
				}
				for _, m := range append(lm[1:], wm...) {
					log.Debugf("Joining %s as worker", m.GetName())
					cliW, err := m.GetEngineAPI()
					if err != nil {
						return err
					}
					err = cliW.SwarmJoin(context.TODO(), swarm.JoinRequest{
						ListenAddr:  listenAddr,
						RemoteAddrs: []string{info.Swarm.RemoteManagers[0].Addr},
						JoinToken:   swarmInfo.JoinTokens.Worker,
					})
					if err != nil {
						return err
					}
				}
			}
			for _, m := range append(lm, wm...) {
				fmt.Println(m.GetConnectionEnv())
				fmt.Println("")
			}
			return nil
		},
	}
	createCmd.Flags().BoolP("debug", "d", false, "enable verbose logging")
	createCmd.Flags().BoolP("no-swarm", "n", false, "skip swarm init and join")
	createCmd.Flags().String("advertise-addr", "", "passed to swarm init")
	createCmd.Flags().String("listen-addr", "0.0.0.0:2377", "passed to swarm init and join")

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
				func() bool { return args[0] == "-" },
				func() (string, error) { return provisionProcessTemplate(args[0]) },
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

	createCmd.AddCommand(infrakit)

	return createCmd
}

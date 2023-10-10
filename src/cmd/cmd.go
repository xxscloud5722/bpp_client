package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/nuwa/bpp.v3/console"
	"github.com/nuwa/bpp.v3/environment"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"os"
)

func Command() []*cobra.Command {
	var packageCmd = &cobra.Command{
		Use:     "package",
		Short:   "Generate App Package Script",
		Example: "package",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.Package()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var pushCmd = &cobra.Command{
		Use:     "push",
		Short:   "Docker Image Push",
		Example: "push",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.DockerPush()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var pullCmd = &cobra.Command{
		Use:     "pull",
		Short:   "Docker Image Pull",
		Example: "pull",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.DockerPull()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var tagCmd = &cobra.Command{
		Use:     "tag",
		Short:   "Docker Image Tag",
		Example: "tag",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.DockerTag()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var removeCmd = &cobra.Command{
		Use:     "remove",
		Short:   "Docker Image Remove",
		Example: "remove",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.DockerRemove()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var releaseCmd = &cobra.Command{
		Use:     "release",
		Short:   "Kubernetes Release Config",
		Example: "release",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.KubernetesRelease()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var sshCmd = &cobra.Command{
		Use:     "ssh",
		Short:   "SSH Release",
		Example: "ssh",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.SSHRelease()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var nacosSyncCmd = &cobra.Command{
		Use:     "nacosSync",
		Short:   "Nacos Config Sync",
		Example: "nacosSync",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.NacosSync()
			if err != nil {
				_ = console.SendMessage(false, fmt.Sprint(err))
				color.Red(fmt.Sprint(err))
				os.Exit(1)
			}
		},
	}

	var environmentCmd = &cobra.Command{
		Use:     "env",
		Short:   "Environment Operate Admin",
		Example: "env list",
	}

	environmentCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all variables",
		Run: func(cmd *cobra.Command, args []string) {
			err := environment.Print(lo.IfF(len(args) > 0, func() string { return args[0] }).Else(""))
			if err != nil {
				color.Red(fmt.Sprint(err))
			}
		},
	})
	environmentCmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get a single variable",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 0 {
				return
			}
			err := environment.PrintByKey(args[0])
			if err != nil {
				color.Red(fmt.Sprint(err))
			}
		},
	})
	environmentCmd.AddCommand(&cobra.Command{
		Use:   "push",
		Short: "Add variables",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				return
			}
			err := environment.Push(args[0], args[1], lo.IfF(len(args) > 2, func() string { return args[2] }).Else(""))
			if err != nil {
				color.Red(fmt.Sprint(err))
			}
		},
	})
	environmentCmd.AddCommand(&cobra.Command{
		Use:   "remove",
		Short: "Remove variables",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 0 {
				return
			}
			err := environment.Remove(args[0])
			if err != nil {
				color.Red(fmt.Sprint(err))
			}
		},
	})

	return []*cobra.Command{
		packageCmd,
		pushCmd,
		pullCmd,
		tagCmd,
		removeCmd,
		releaseCmd,
		sshCmd,
		nacosSyncCmd,
		environmentCmd,
	}
}

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
	var releaseCmd = &cobra.Command{
		Use:     "release",
		Short:   "Kubernetes Release Config",
		Example: "release",
		Run: func(cmd *cobra.Command, args []string) {
			err := console.KubernetesRelease()
			if err != nil {
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
		releaseCmd,
		nacosSyncCmd,
		environmentCmd,
	}
}

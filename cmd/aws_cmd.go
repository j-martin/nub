package cmd

import (
	"github.com/j-martin/bub/core"
	"github.com/j-martin/bub/integrations/aws"
	"github.com/urfave/cli"
	"log"
)

func buildEC2Cmd(cfg *core.Configuration, manifest *core.Manifest) cli.Command {
	jump := "jump"
	all := "all"
	output := "output"

	return cli.Command{
		Name: "ec2",
		Usage: "EC2 related related actions. The commands 'bash', 'exec', " +
			"'jstack' and 'jmap' will be executed inside the container.",
		ArgsUsage: "[INSTANCE_NAME] [COMMAND ...]",
		Aliases:   []string{"e"},
		Flags: []cli.Flag{
			cli.BoolFlag{Name: jump, Usage: "Use the environment jump host."},
			cli.BoolFlag{Name: all, Usage: "Execute the command on all the instance matched."},
			cli.BoolFlag{Name: output, Usage: "Saves the stdout of the command to a file."},
		},
		Action: func(c *cli.Context) error {
			var (
				name string
				args []string
			)
			if c.NArg() > 0 {
				name = c.Args().Get(0)
			} else if manifest.Name != "" {
				log.Printf("Manifest found. Using '%v'", name)
				name = manifest.Name
			}
			if c.NArg() > 1 {
				args = c.Args()[1:]
			}
			return aws.ConnectToInstance(aws.ConnectionParams{
				Configuration: cfg,
				Filter:        name,
				Output:        c.Bool(output),
				All:           c.Bool(all),
				UseJumpHost:   c.Bool(jump),
				Args:          args},
			)
		},
	}
}

func buildRDSCmd(cfg *core.Configuration) cli.Command {
	return cli.Command{
		Name:    "rds",
		Usage:   "RDS actions.",
		Aliases: []string{"r"},
		Action: func(c *cli.Context) error {
			return aws.GetRDS(cfg).ConnectToRDSInstance(c.Args().First(), c.Args().Tail())
		},
	}
}

func buildR53Cmd() cli.Command {
	return cli.Command{
		Name:    "route53",
		Usage:   "R53 actions.",
		Aliases: []string{"53"},
		Action: func(c *cli.Context) error {
			return aws.ListAllRecords(c.Args().First())
		},
	}
}

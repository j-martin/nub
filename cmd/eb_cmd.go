package cmd

import (
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/aws"
	"github.com/urfave/cli"
	"log"
	"os"
	"strings"
	"time"
)

func getRegion(environment string, cfg *core.Configuration, c *cli.Context) string {
	region := c.String("region")
	if region == "" {
		prefix := strings.Split(environment, "-")[0]
		for _, i := range cfg.AWS.Environments {
			if i.Prefix == prefix {
				return i.Region
			}
		}
		return cfg.AWS.Regions[0]
	}
	return region
}

func buildEBCmd(cfg *core.Configuration, manifest *core.Manifest) cli.Command {
	return cli.Command{
		Name:    "beanstalk",
		Usage:   "Elasticbeanstalk actions. If no sub-command specified, lists the environements.",
		Aliases: []string{"eb"},
		Action: func(c *cli.Context) error {
			aws.ListEnvironments(cfg)
			return nil
		},
		Subcommands: buildEBCmds(cfg, manifest),
	}
}

func buildEBCmds(cfg *core.Configuration, manifest *core.Manifest) []cli.Command {
	region := "region"
	reverse := "reverse"
	all := "all"

	return []cli.Command{
		{
			Name:    "environments",
			Aliases: []string{"env"},
			Usage:   "List enviroments and their states.",
			Flags: []cli.Flag{
				cli.StringFlag{Name: region},
			},
			Action: func(c *cli.Context) error {
				aws.ListEnvironments(cfg)
				return nil
			},
		},
		{
			Name:      "events",
			Aliases:   []string{"e"},
			Usage:     "List events for all environments.",
			UsageText: "[ENVIRONMENT_NAME] Optional filter by environment name.",
			Flags: []cli.Flag{
				cli.StringFlag{Name: region},
				cli.BoolFlag{Name: reverse},
			},
			Action: func(c *cli.Context) error {
				environment := ""
				if c.NArg() > 0 {
					environment = c.Args().Get(0)
				} else if manifest.Name != "" {
					environment = "pro-" + manifest.Name
					log.Printf("Manifest found. Using '%v'", environment)
				}
				aws.ListEvents(getRegion(environment, cfg, c), environment, time.Time{}, c.Bool(reverse), true, false)
				return nil
			},
		},
		{
			Name:      "ready",
			Aliases:   []string{"r"},
			Usage:     "Wait for environment to be ready.",
			UsageText: "ENVIRONMENT_NAME",
			Flags: []cli.Flag{
				cli.StringFlag{Name: region},
			},
			Action: func(c *cli.Context) error {
				environment := ""
				if c.NArg() > 0 {
					environment = c.Args().Get(0)
				} else if manifest.Name != "" {
					environment = "pro-" + manifest.Name
					log.Printf("Manifest found. Using '%v'", environment)
				}
				aws.EnvironmentIsReady(getRegion(environment, cfg, c), environment, true)
				return nil
			},
		},
		{
			Name:      "settings",
			Aliases:   []string{"s"},
			Usage:     "List Environment settings",
			UsageText: "ENVIRONMENT_NAME",
			Flags: []cli.Flag{
				cli.StringFlag{Name: region},
				cli.BoolFlag{Name: all, Usage: "Display all settings, not just environment variables."},
			},
			Action: func(c *cli.Context) error {
				environment := ""
				if c.NArg() > 0 {
					environment = c.Args().Get(0)
				} else if manifest.Name != "" {
					environment = "pro-" + manifest.Name
					log.Printf("Manifest found. Using '%v'", environment)
				}
				aws.DescribeEnvironment(getRegion(environment, cfg, c), environment, c.Bool(all))
				return nil
			},
		},
		{
			Name:      "versions",
			Aliases:   []string{"v"},
			Usage:     "List all versions available.",
			ArgsUsage: "[APPLICATION_NAME] Optional, limits the versions to the application name.",
			Flags: []cli.Flag{
				cli.StringFlag{Name: region},
			},
			Action: func(c *cli.Context) error {
				application := ""
				if c.NArg() > 0 {
					application = c.Args().Get(0)
				} else if manifest.Name != "" {
					application = manifest.Name
					log.Printf("Manifest found. Using '%v'", application)
				}

				aws.ListApplicationVersions(getRegion(application, cfg, c), application)
				return nil
			},
		},
		{
			Name:      "deploy",
			Aliases:   []string{"d"},
			Usage:     "Deploy version to an environment.",
			ArgsUsage: "[ENVIRONMENT_NAME] [VERSION]",
			Flags: []cli.Flag{
				cli.StringFlag{Name: region},
			},
			Action: func(c *cli.Context) error {
				environment := ""
				if c.NArg() > 0 {
					environment = c.Args().Get(0)
				} else if manifest.Name != "" {
					environment = "pro-" + manifest.Name
					log.Printf("Manifest found. Using '%v'", environment)
				} else {
					log.Fatal("Environment required. Stopping.")
					os.Exit(1)
				}

				region := getRegion(environment, cfg, c)

				if c.NArg() < 2 {
					aws.ListApplicationVersions(region, aws.GetApplication(environment))
					log.Println("Version required. Specify one of the application versions above.")
					os.Exit(2)
				}
				version := c.Args().Get(1)
				aws.DeployVersion(region, environment, version)
				return nil
			},
		},
	}
}

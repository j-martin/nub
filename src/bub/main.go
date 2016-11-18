package main

import (
	"fmt"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "bub"
	app.Usage = "A tool for all your Bench related things."
	app.Version = "0.6.4"
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		{
			Name:  "setup",
			Usage: "Setup bub on your machine.",
			Action: func(c *cli.Context) error {
				Setup()
				return nil
			},
		},
		{
			Name:    "repository",
			Aliases: []string{"r"},
			Usage:   "Synchronize the all the active repositories.",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "force", Usage: "Skips the confirmation prompt."},
			},
			Action: func(c *cli.Context) error {
				message := `

STOP!

This command will clone and/or Update all 'active' Bench repositories.
Existing work will be stashed and pull the master branch. Your work won't be lost, but be careful.
Please make sure you are in the directory where you store your repos and not a specific repo.

Continue?`
				if c.Bool("force") || askForConfirmation(message) {
					SyncRepositories()
				} else {
					os.Exit(1)
				}
				return nil
			},
		},
		{
			Name:    "manifest",
			Aliases: []string{"m"},
			Usage:   "Manifest related actions.",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "List all manifests.",
					Flags: []cli.Flag{
						cli.BoolFlag{Name: "full", Usage: "Display all information, including readmes and changelogs."},
						cli.BoolFlag{Name: "active", Usage: "Display only active projects."},
					},
					Action: func(c *cli.Context) error {
						manifests := GetAllManifests()
						for _, m := range manifests {
							if !c.Bool("full") {
								m.Readme = ""
								m.ChangeLog = ""
							}
							if !c.Bool("active") || (c.Bool("active") && m.Active) {
								yml, _ := yaml.Marshal(m)
								fmt.Println(string(yml))
							}
						}
						return nil
					},
				},
				{
					Name:    "create",
					Aliases: []string{"c"},
					Usage:   "Creates a base manifest.",
					Action: func(c *cli.Context) error {
						CreateManifest()
						return nil
					},
				},
				{
					Name:    "update",
					Aliases: []string{"u"},
					Usage:   "Updates/uploads the manifest to the database.",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "artifact-version"},
					},
					Action: func(c *cli.Context) error {
						m := BuildManifest(c.String("artifact-version"))
						StoreManifest(m)
						UpdateDocumentation(m)
						return nil
					},
				},
				{
					Name:    "validate",
					Aliases: []string{"v"},
					Usage:   "Validates the manifest.",
					Action: func(c *cli.Context) error {
						//TODO: Build proper validation
						yml, _ := yaml.Marshal(BuildManifest(""))
						log.Println(string(yml))
						return nil
					},
				},
			},
		},
		{
			Name:      "ec2",
			Usage:     "EC2 related related actions.",
			ArgsUsage: "[INSTANCE_NAME] [COMMAND ...]",
			Aliases:   []string{"e"},
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "all", Usage: "Execute the command on all the instance matchrd."},
				cli.BoolFlag{Name: "output", Usage: "Saves the stdout of the command to a file."},
			},
			Action: func(c *cli.Context) error {
				var (
					name string
					args []string
				)

				if c.NArg() > 0 {
					name = c.Args().Get(0)
				}

				if c.NArg() > 1 {
					args = c.Args()[1:]
				}

				ConnectToInstance(ConnectionParams{name, c.Bool("output"), c.Bool("all"), args})
				return nil
			},
		},
		{
			Name:    "elasticbeanstalk",
			Usage:   "Elasticbeanstalk actions. If no sub-action specified, lists the environements.",
			Aliases: []string{"eb"},
			Action: func(c *cli.Context) error {
				ListEnvironments()
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "environments",
					Aliases: []string{"env"},
					Usage:   "List enviroments and their states.",
					Action: func(c *cli.Context) error {
						ListEnvironments()
						return nil
					},
				},
				{
					Name:    "events",
					Aliases: []string{"e"},
					Usage:   "List events for all environments.",
					Action: func(c *cli.Context) error {
						ListEvents()
						return nil
					},
				},
			},
		},
		{
			Name:    "github",
			Usage:   "GitHub related actions.",
			Aliases: []string{"gh"},
			Subcommands: []cli.Command{
				{
					Name:    "repo",
					Aliases: []string{"r"},
					Usage:   "Open repo in your browser.",
					Action: func(c *cli.Context) error {
						OpenGH(BuildManifest(""), "")
						return nil
					},
				},
				{
					Name:    "issues",
					Aliases: []string{"i"},
					Usage:   "Open issues list in your browser.",
					Action: func(c *cli.Context) error {
						OpenGH(BuildManifest(""), "issues")
						return nil
					},
				},
				{
					Name:    "branches",
					Aliases: []string{"b"},
					Usage:   "Open branches list in your browser.",
					Action: func(c *cli.Context) error {
						OpenGH(BuildManifest(""), "branches")
						return nil
					},
				},
				{
					Name:    "pr",
					Aliases: []string{"p"},
					Usage:   "Open Pull Request list in your browser.",
					Action: func(c *cli.Context) error {
						OpenGH(BuildManifest(""), "pulls")
						return nil
					},
				},
			},
		},
		{
			Name:    "jenkins",
			Usage:   "Jenkins related actions.",
			Aliases: []string{"j"},
			Action: func(c *cli.Context) error {
				OpenJenkins(BuildManifest(""), "")
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "console",
					Aliases: []string{"c"},
					Usage:   "Opens the console of the last build.",
					Action: func(c *cli.Context) error {
						OpenJenkins(BuildManifest(""), "job/master/lastBuild/consoleFull")
						return nil
					},
				},
			},
		},
		{
			Name:    "splunk",
			Usage:   "Open the service production logs.",
			Aliases: []string{"s"},
			Action: func(c *cli.Context) error {
				m := BuildManifest("")
				OpenSplunk(m, false)
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "staging",
					Aliases: []string{"s"},
					Usage:   "Open the service staging logs.",
					Action: func(c *cli.Context) error {
						m := BuildManifest("")
						OpenSplunk(m, true)
						return nil
					},
				},
			},
		},
		{
			Name:    "docs",
			Usage:   "Documentation related actions.",
			Aliases: []string{"d"},
			Action: func(c *cli.Context) error {
				m := BuildManifest("")
				base := "https://example.atlassian.net/wiki/display/dev/"
				OpenURI(base + m.Name)
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "raml",
					Usage:   "Opens raml file on GitHub.",
					Aliases: []string{"r"},
					Action: func(c *cli.Context) error {
						base := "https://github.com/BenchLabs/bench-raml/tree/master/specs/"
						OpenURI(base + BuildManifest("").Name + ".raml")
						return nil
					},
				},
			},
		},
		{
			Name:    "circle",
			Usage:   "Opens the repo's CircleCI test results.",
			Aliases: []string{"c"},
			Action: func(c *cli.Context) error {
				OpenCircle(BuildManifest(""), false)
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "circle",
					Usage:   "Opens the result for the current branch.",
					Aliases: []string{"c"},
					Action: func(c *cli.Context) error {
						OpenCircle(BuildManifest(""), true)
						return nil
					},
				},
			},
		},
	}

	app.Run(os.Args)
}

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

func getRegion(environment string, cfg Configuration, c *cli.Context) string {

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

func main() {
	cfg := loadConfiguration()
	manifest, manifestErr := loadManifest("")

	app := cli.NewApp()
	app.Name = "bub"
	app.Usage = "A tool for all your Bench related needs."
	app.Version = "0.19.6"
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		{
			Name:  "setup",
			Usage: "Setup bub on your machine.",
			Action: func(c *cli.Context) error {
				setup()
				return nil
			},
		},
		{
			Name:  "update",
			Usage: "Update the bub command to the latest release",
			Action: func(c *cli.Context) error {
				path := S3path{
					Region: cfg.Updates.Region,
					Bucket: cfg.Updates.Bucket,
					Path:   cfg.Updates.Prefix,
				}
				obj, err := latestRelease(path)
				if err != nil {
					return err
				}
				path.Path = *obj.Key
				return updateBub(path)
			},
		},
		{
			Name:  "config",
			Usage: "Edit your bub config",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "show-default", Usage: "Show default config for reference"},
			},
			Action: func(c *cli.Context) error {
				if c.Bool("show-default") {
					print(config)
				} else {
					editConfiguration()
				}
				return nil
			},
		},
		{
			Name:    "repository",
			Usage:   "Repository related actions",
			Aliases: []string{"r"},
			Subcommands: []cli.Command{
				{
					Name:  "synchronize",
					Usage: "Synchronize the all the active repositories.",
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
					Name:    "pending",
					Aliases: []string{"p"},
					Usage:   "List diff between the previous version and the next one.",
					Flags: []cli.Flag{
						cli.BoolFlag{Name: "slack-format", Usage: "Format the result for slack."},
						cli.BoolFlag{Name: "slack-no-at", Usage: "Do not add @person at the end."},
						cli.BoolFlag{Name: "no-fetch", Usage: "Do not fetch tags."},
					},
					Action: func(c *cli.Context) error {
						if !c.Bool("no-fetch") {
							FetchTags()
						}
						previousVersion := "production"
						if len(c.Args()) > 0 {
							previousVersion = c.Args().Get(0)
						}
						nextVersion := "HEAD"
						if len(c.Args()) > 1 {
							nextVersion = c.Args().Get(1)
						}
						PendingChanges(cfg, manifest, previousVersion, nextVersion, c.Bool("slack-format"), c.Bool("slack-no-at"))
						return nil
					},
				},
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
						cli.BoolFlag{Name: "name", Usage: "Display only the project names."},
						cli.BoolFlag{Name: "service", Usage: "Display only the services projects."},
						cli.BoolFlag{Name: "lib", Usage: "Display only the library projects."},
						cli.StringFlag{Name: "lang", Usage: "Display only projects matching the language"},
					},
					Action: func(c *cli.Context) error {
						manifests := GetAllManifests()
						for _, m := range manifests {
							if !c.Bool("full") {
								m.Readme = ""
								m.ChangeLog = ""
							}

							if c.Bool("active") && !m.Active {
								continue
							}

							if c.Bool("service") && !isSameType(m, "service") {
								continue
							}

							if c.Bool("lib") && !isSameType(m, "library") {
								continue
							}

							if c.String("lang") != "" && m.Language != c.String("lang") {
								continue
							}

							if c.Bool("name") {
								fmt.Println(m.Name)
							} else {
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
						createManifest()
						return nil
					},
				},
				{
					Name:    "graph",
					Aliases: []string{"g"},
					Usage:   "Creates dependency graph from manifests.",
					Action: func(c *cli.Context) error {
						generateGraphs()
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
						if manifestErr != nil {
							log.Fatal(manifestErr)
							os.Exit(1)
						}
						manifest.Version = c.String("artifact-version")
						StoreManifest(manifest)
						updateDocumentation(cfg, manifest)
						return nil
					},
				},
				{
					Name:    "validate",
					Aliases: []string{"v"},
					Usage:   "Validates the manifest.",
					Action: func(c *cli.Context) error {
						//TODO: Build proper validation
						if manifestErr != nil {
							log.Fatal(manifestErr)
							os.Exit(1)
						}
						manifest.Version = c.String("artifact-version")
						yml, _ := yaml.Marshal(manifest)
						log.Println(string(yml))
						return nil
					},
				},
			},
		},
		{
			Name: "ec2",
			Usage: "EC2 related related actions. The commands 'bash', 'exec', " +
				"'jstack' and 'jmap' will be executed inside the container.",
			ArgsUsage: "[INSTANCE_NAME] [COMMAND ...]",
			Aliases:   []string{"e"},
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "jump", Usage: "Use the environment jump host."},
				cli.BoolFlag{Name: "all", Usage: "Execute the command on all the instance matched."},
				cli.BoolFlag{Name: "output", Usage: "Saves the stdout of the command to a file."},
			},
			Action: func(c *cli.Context) error {
				var (
					name string
					args []string
				)

				if c.NArg() > 0 {
					name = c.Args().Get(0)
				} else if manifestErr == nil {
					log.Printf("Manifest found. Using '%v'", name)
					name = manifest.Name
				}

				if c.NArg() > 1 {
					args = c.Args()[1:]
				}

				ConnectToInstance(ConnectionParams{cfg, name, c.Bool("output"), c.Bool("all"), c.Bool("jump"), args})
				return nil
			},
		},
		{
			Name:    "rds",
			Usage:   "RDS actions.",
			Aliases: []string{"r"},
			Action: func(c *cli.Context) error {
				ConnectToRDSInstance(cfg, c.Args().First(), c.Args().Tail())
				return nil
			},
		},
		{
			Name:    "elasticbeanstalk",
			Usage:   "Elasticbeanstalk actions. If no sub-action specified, lists the environements.",
			Aliases: []string{"eb"},
			Flags: []cli.Flag{
				cli.StringFlag{Name: "region"},
			},
			Action: func(c *cli.Context) error {
				ListEnvironments(cfg)
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "environments",
					Aliases: []string{"env"},
					Usage:   "List enviroments and their states.",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "region"},
					},
					Action: func(c *cli.Context) error {
						ListEnvironments(cfg)
						return nil
					},
				},
				{
					Name:      "events",
					Aliases:   []string{"e"},
					Usage:     "List events for all environments.",
					UsageText: "[ENVIRONMENT_NAME] Optional filter by environment name.",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "region"},
						cli.BoolFlag{Name: "reverse"},
					},
					Action: func(c *cli.Context) error {
						environment := ""
						if c.NArg() > 0 {
							environment = c.Args().Get(0)
						} else if manifestErr == nil {
							environment = "prod-" + manifest.Name
							log.Printf("Manifest found. Using '%v'", environment)
						}
						ListEvents(getRegion(environment, cfg, c), environment, time.Time{}, c.Bool("reverse"), true, false)
						return nil
					},
				},
				{
					Name:      "ready",
					Aliases:   []string{"r"},
					Usage:     "Wait for environment to be ready.",
					UsageText: "ENVIRONMENT_NAME",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "region"},
					},
					Action: func(c *cli.Context) error {
						environment := ""
						if c.NArg() > 0 {
							environment = c.Args().Get(0)
						} else if manifestErr == nil {
							environment = "prod-" + manifest.Name
							log.Printf("Manifest found. Using '%v'", environment)
						}
						EnvironmentIsReady(getRegion(environment, cfg, c), environment, true)
						return nil
					},
				},
				{
					Name:      "settings",
					Aliases:   []string{"s"},
					Usage:     "List Environment settings",
					UsageText: "ENVIRONMENT_NAME",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "region"},
						cli.BoolFlag{Name: "all", Usage: "Display all settings, not just environment variables."},
					},
					Action: func(c *cli.Context) error {
						environment := ""
						if c.NArg() > 0 {
							environment = c.Args().Get(0)
						} else if manifestErr == nil {
							environment = "prod-" + manifest.Name
							log.Printf("Manifest found. Using '%v'", environment)
						}
						DescribeEnvironment(getRegion(environment, cfg, c), environment, c.Bool("all"))
						return nil
					},
				},
				{
					Name:      "versions",
					Aliases:   []string{"v"},
					Usage:     "List all versions available.",
					ArgsUsage: "[APPLICATION_NAME] Optional, limits the versions to the application name.",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "region"},
					},
					Action: func(c *cli.Context) error {
						application := ""
						if c.NArg() > 0 {
							application = c.Args().Get(0)
						} else if manifestErr == nil {
							application = manifest.Name
							log.Printf("Manifest found. Using '%v'", application)
						}

						ListApplicationVersions(getRegion(application, cfg, c), application)
						return nil
					},
				},
				{
					Name:      "deploy",
					Aliases:   []string{"d"},
					Usage:     "Deploy version to an environment.",
					ArgsUsage: "[ENVIRONMENT_NAME] [VERSION]",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "region"},
					},
					Action: func(c *cli.Context) error {
						environment := ""
						if c.NArg() > 0 {
							environment = c.Args().Get(0)
						} else if manifestErr == nil {
							environment = "prod-" + manifest.Name
							log.Printf("Manifest found. Using '%v'", environment)
						} else {
							log.Fatal("Environment required. Stopping.")
							os.Exit(1)
						}

						region := getRegion(environment, cfg, c)

						if c.NArg() < 2 {
							ListApplicationVersions(region, GetApplication(environment))
							log.Println("Version required. Specify one of the application versions above.")
							os.Exit(2)
						}
						version := c.Args().Get(1)
						DeployVersion(region, environment, version)
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
						openGH(manifest, "")
						return nil
					},
				},
				{
					Name:    "issues",
					Aliases: []string{"i"},
					Usage:   "Open issues list in your browser.",
					Action: func(c *cli.Context) error {
						openGH(manifest, "issues")
						return nil
					},
				},
				{
					Name:    "branches",
					Aliases: []string{"b"},
					Usage:   "Open branches list in your browser.",
					Action: func(c *cli.Context) error {
						openGH(manifest, "branches")
						return nil
					},
				},
				{
					Name:    "pr",
					Aliases: []string{"p"},
					Usage:   "Open Pull Request list in your browser.",
					Action: func(c *cli.Context) error {
						openGH(manifest, "pulls")
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
				openJenkins(manifest, "")
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "console",
					Aliases: []string{"c"},
					Usage:   "Opens the (web) console of the last build of master.",
					Action: func(c *cli.Context) error {
						openJenkins(manifest, "lastBuild/consoleFull")
						return nil
					},
				},
				{
					Name:    "jobs",
					Aliases: []string{"j"},
					Usage:   "Shows the console output of the last build.",
					Action: func(c *cli.Context) error {
						showConsoleOutput(cfg, manifest)
						return nil
					},
				},
				{
					Name:    "artifacts",
					Aliases: []string{"a"},
					Usage:   "Get the previous build's artifacts.",
					Action: func(c *cli.Context) error {
						getArtifacts(cfg, manifest)
						return nil
					},
				},
				{
					Name:    "build",
					Aliases: []string{"b"},
					Flags: []cli.Flag{
						cli.BoolFlag{Name: "no-wait", Usage: "Do not wait for the job to be completed."},
						cli.BoolFlag{Name: "force", Usage: "Trigger job regardless if a build running."},
					},
					Usage: "Trigger build of the current branch.",
					Action: func(c *cli.Context) error {
						buildJob(cfg, manifest, c.Bool("no-wait"), c.Bool("force"))
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
				openSplunk(manifest, false)
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "staging",
					Aliases: []string{"s"},
					Usage:   "Open the service staging logs.",
					Action: func(c *cli.Context) error {
						openSplunk(manifest, true)
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
				base := "https://example.atlassian.net/wiki/display/dev/"
				openURI(base + manifest.Name)
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "raml",
					Usage:   "Opens raml file on GitHub.",
					Aliases: []string{"r"},
					Action: func(c *cli.Context) error {
						base := "https://github.com/BenchLabs/bench-raml/tree/master/specs/"
						openURI(base + manifest.Name + ".raml")
						return nil
					},
				},
			},
		},
		{
			Name:    "circle",
			Usage:   "CircleCI related actions",
			Aliases: []string{"c"},
			Action: func(c *cli.Context) error {
				openCircle(manifest, false)
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:    "trigger",
					Usage:   "Trigger the current branch of the current repo and wait for success.",
					Aliases: []string{"t"},
					Action: func(c *cli.Context) error {
						triggerAndWaitForSuccess(cfg, manifest)
						return nil
					},
				},
				{
					Name:    "circle",
					Usage:   "Opens the result for the current branch.",
					Aliases: []string{"b"},
					Action: func(c *cli.Context) error {
						openCircle(manifest, true)
						return nil
					},
				},
			},
		},
	}

	app.Run(os.Args)
}

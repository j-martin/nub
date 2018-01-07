package cmd

import (
	"fmt"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/atlassian"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

func buildManifestCmds(cfg *core.Configuration) []cli.Command {
	return []cli.Command{
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
				manifests := core.GetManifestRepository().GetAllManifests()
				for _, m := range manifests {
					if !c.Bool("full") {
						m.Readme = ""
						m.ChangeLog = ""
					}

					if c.Bool("active") && !m.Active {
						continue
					}

					if c.Bool("service") && !core.IsSameType(m, "service") {
						continue
					}

					if c.Bool("lib") && !core.IsSameType(m, "library") {
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
				core.CreateManifest()
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
				manifest, err := core.LoadManifest("")
				if err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
				manifest.Version = c.String("artifact-version")
				core.GetManifestRepository().StoreManifest(manifest)
				return atlassian.MustInitConfluence(cfg).UpdateDocumentation(manifest)
			},
		},
		{
			Name:    "validate",
			Aliases: []string{"v"},
			Usage:   "Validates the manifest.",
			Action: func(c *cli.Context) error {
				//TODO: Build proper validation
				manifest, err := core.LoadManifest("")
				if err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
				manifest.Version = c.String("artifact-version")
				yml, _ := yaml.Marshal(manifest)
				log.Println(string(yml))
				return nil
			},
		},
	}
}

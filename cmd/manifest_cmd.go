package cmd

import (
	"github.com/j-martin/nub/core"
	"github.com/j-martin/nub/integrations/github"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

func buildManifestCmds(cfg *core.Configuration) []cli.Command {
	return []cli.Command{
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
			Name:    "validate",
			Aliases: []string{"v"},
			Usage:   "Validates the manifest.",
			Action: func(c *cli.Context) error {
				//TODO: Build proper validation
				manifest, err := core.LoadManifest()
				if err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
				manifest.Version = c.String("artifact-version")
				err = github.MustInitGitHub(cfg).PopulateOwners(manifest)
				if err != nil {
					log.Print(err)
				}
				yml, _ := yaml.Marshal(manifest)
				log.Println(string(yml))
				return nil
			},
		},
	}
}

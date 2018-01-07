package cmd

import (
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations"
	"github.com/urfave/cli"
)

func buildGitHubCmds(cfg *core.Configuration, manifest *core.Manifest) []cli.Command {
	return []cli.Command{
		{
			Name:    "repo",
			Aliases: []string{"r"},
			Usage:   "Open repo in your browser.",
			Action: func(c *cli.Context) error {
				return integrations.MustInitGitHub(cfg).OpenPage(manifest)
			},
		},
		{
			Name:    "issues",
			Aliases: []string{"i"},
			Usage:   "Open issues list in your browser.",
			Action: func(c *cli.Context) error {
				return integrations.MustInitGitHub(cfg).OpenPage(manifest, "issues")
			},
		},
		{
			Name:    "branches",
			Aliases: []string{"b"},
			Usage:   "Open branches list in your browser.",
			Action: func(c *cli.Context) error {
				return integrations.MustInitGitHub(cfg).OpenPage(manifest, "branches")
			},
		},
		{
			Name:    "pr",
			Aliases: []string{"p"},
			Usage:   "Open Pull Request list in your browser.",
			Action: func(c *cli.Context) error {
				return integrations.MustInitGitHub(cfg).OpenPage(manifest, "pulls")
			},
		},
		{
			Name:  "stale-branches",
			Usage: "Open repo in your browser.",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "max-age", Value: "30"},
			},
			Action: func(c *cli.Context) error {
				return integrations.MustInitGitHub(cfg).ListBranches(c.Int("max-age"))
			},
		},
	}
}

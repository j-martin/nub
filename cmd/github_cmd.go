package cmd

import (
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/github"
	"github.com/urfave/cli"
)

func buildGitHubCmds(cfg *core.Configuration, manifest *core.Manifest) []cli.Command {
	maxAge := "max-age"
	closed := "closed"
	role := "role"
	openAll := "open-all"
	return []cli.Command{
		{
			Name:    "repo",
			Aliases: []string{"r"},
			Usage:   "Open repo in your browser.",
			Action: func(c *cli.Context) error {
				return github.MustInitGitHub(cfg).OpenPage(manifest)
			},
		},
		{
			Name:    "issues",
			Aliases: []string{"i"},
			Usage:   "Open issues list in your browser.",
			Action: func(c *cli.Context) error {
				return github.MustInitGitHub(cfg).OpenPage(manifest, "issues")
			},
		},
		{
			Name:    "branches",
			Aliases: []string{"b"},
			Usage:   "Open branches list in your browser.",
			Action: func(c *cli.Context) error {
				return github.MustInitGitHub(cfg).OpenPage(manifest, "branches")
			},
		},
		{
			Name:    "pr",
			Aliases: []string{"p"},
			Usage:   "Open Pull Request list in your browser.",
			Action: func(c *cli.Context) error {
				return github.MustInitGitHub(cfg).OpenPage(manifest, "pulls")
			},
		},
		{
			Name:    "list-pr",
			Aliases: []string{"l"},
			Usage:   "List PR assigned to you.",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: closed, Usage: "Show closed PRs."},
				cli.StringFlag{Name: role, Usage: "Filter by role. E.g. 'involved', 'review-requested', etc. Default: 'author'"},
				cli.BoolFlag{Name: openAll, Usage: "Open all PRs in the browser."},
			},
			Action: func(c *cli.Context) error {
				return github.MustInitGitHub(cfg).SearchIssues("pr", c.String(role), c.Bool(closed), c.Bool(openAll))
			},
		},
		{
			Name:    "list-pr-reviews",
			Aliases: []string{"lr"},
			Usage:   "List PR assigned to you.",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: openAll, Usage: "Open all PRs in the browser."},
			},
			Action: func(c *cli.Context) error {
				return github.MustInitGitHub(cfg).SearchIssues("pr", "review-requested", false, c.Bool(openAll))
			},
		},
		{
			Name:  "stale-branches",
			Usage: "Open repo in your browser.",
			Flags: []cli.Flag{
				cli.StringFlag{Name: maxAge, Value: "30"},
			},
			Action: func(c *cli.Context) error {
				return github.MustInitGitHub(cfg).ListBranches(c.Int(maxAge))
			},
		},
	}
}

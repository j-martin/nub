package cmd

import (
	"log"

	"github.com/j-martin/nub/core"
	"github.com/j-martin/nub/integrations/atlassian"
	"github.com/j-martin/nub/integrations/github"
	"github.com/urfave/cli"
)

func buildSetupCmd() cli.Command {
	resetCredentials := "reset-credentials"
	return cli.Command{
		Name:  "setup",
		Usage: "Setup nub on your machine.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: resetCredentials, Usage: "Prompt you to re-enter credentials."},
		},
		Action: func(c *cli.Context) error {
			core.MustSetupConfig()
			// Reloading the config
			cfg, _ := core.LoadConfiguration()
			cfg.ResetCredentials = c.Bool(resetCredentials)
			atlassian.MustSetupJIRA(cfg)
			atlassian.MustSetupConfluence(cfg)
			github.MustSetupGitHub(cfg)
			log.Println("Done.")
			return nil
		},
	}
}

func buildRepositoryCmds(cfg *core.Configuration, manifest *core.Manifest) []cli.Command {
	slackFormat := "slack-format"
	noSlackAt := "slack-no-at"
	noFetch := "no-fetch"
	return []cli.Command{
		{
			Name:    "pending",
			Aliases: []string{"p"},
			Usage:   "List diff between the previous version and the next one.",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: slackFormat, Usage: "Format the result for slack."},
				cli.BoolFlag{Name: noSlackAt, Usage: "Do not add @person at the end."},
				cli.BoolFlag{Name: noFetch, Usage: "Do not fetch tags."},
			},
			Action: func(c *cli.Context) error {
				if !c.Bool(noFetch) {
					err := core.InitGit().FetchTags()
					if err != nil {
						return err
					}
				}
				previousVersion := "production"
				if len(c.Args()) > 0 {
					previousVersion = c.Args().Get(0)
				}
				nextVersion := "HEAD"
				if len(c.Args()) > 1 {
					nextVersion = c.Args().Get(1)
				}
				core.InitGit().PendingChanges(cfg, manifest, previousVersion, nextVersion, c.Bool(slackFormat), c.Bool(noSlackAt))
				return nil
			},
		},
	}
}

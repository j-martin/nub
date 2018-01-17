package cmd

import (
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/atlassian"
	"github.com/benchlabs/bub/integrations/github"
	"github.com/benchlabs/bub/utils"
	"github.com/urfave/cli"
	"log"
	"os"
)

func buildWorkflowCmds(cfg *core.Configuration, manifest *core.Manifest) []cli.Command {
	transition := "t"
	noOperation := "noop"
	compare := "compare-only"
	return []cli.Command{
		buildJIRAOpenBoardCmd(cfg),
		buildJIRAClaimIssueCmd(cfg),
		buildJIRAOpenIssueCmd(cfg),
		buildJIRAViewIssueCmd(cfg),
		buildJIRACommentOnIssuesCmd(cfg),
		buildJIRAListAssignedIssuesCmd(cfg),
		{
			Name:    "new-branch",
			Aliases: []string{"n", "new"},
			Usage:   "Checkout a new branch based on JIRA issues assigned to you.",
			Action: func(c *cli.Context) error {
				return atlassian.MustInitJIRA(cfg).CreateBranchFromAssignedIssue()
			},
		},
		{
			Name:    "checkout-branch",
			Aliases: []string{"ch", "br"},
			Usage:   "Checkout an existing branch.",
			Action: func(c *cli.Context) error {
				return core.InitGit().CheckoutBranch()
			},
		},
		{
			Name:    "commit",
			Aliases: []string{"c"},
			Usage:   "MESSAGE [OPTS]...",
			Action: func(c *cli.Context) error {
				if len(c.Args()) < 1 {
					log.Fatal("Must pass commit message.")
				}
				core.InitGit().CommitWithIssueKey(cfg, c.Args().Get(0), c.Args().Tail())
				return nil
			},
		},
		{
			Name:    "pull-request",
			Aliases: []string{"pr"},
			Usage:   "Creates a PR for the current branch.",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: compare, Usage: "Open only the compare page (PR creation page)."},
				cli.BoolFlag{Name: transition, Usage: "Transition the issue to review."},
			},
			Action: func(c *cli.Context) error {
				if c.Bool(compare) {
					core.MustInitGit(".").MustPush(cfg)
					return github.MustInitGitHub(cfg).OpenCompareBranchPage(manifest)
				}
				var title, body string
				if len(c.Args()) > 0 {
					title = c.Args().Get(0)
				}
				if len(c.Args()) > 1 {
					body = c.Args().Get(1)
				}
				return MustInitWorkflow(cfg, manifest).CreatePR(title, body, c.Bool("transition"))
			},
		},
		buildJIRATransitionIssueCmd(cfg),
		{
			Name:    "log",
			Aliases: []string{"l"},
			Usage:   "Show git log and open PR, JIRA ticket, etc.",
			Action: func(c *cli.Context) error {
				return MustInitWorkflow(cfg, manifest).Log()
			},
		},
		{
			Name:    "mass",
			Aliases: []string{"m"},
			Usage:   "Mass repo changes. EXPERIMENTAL",
			Subcommands: []cli.Command{
				{
					Name:    "start",
					Aliases: []string{"s"},
					Usage:   "Clean the repository, checkout master, pull and create new branch.",
					Action: func(c *cli.Context) error {
						if !utils.AskForConfirmation("You will lose existing changes.") {
							os.Exit(1)
						}
						return MustInitWorkflow(cfg, manifest).MassStart()
					},
				},
				{
					Name:    "done",
					Aliases: []string{"d"},
					Usage:   "Commit changes and create PRs. To be used after running '... start' and you made your changes.",
					Flags: []cli.Flag{
						cli.BoolFlag{Name: noOperation, Usage: "Do not do any actions."},
					},
					Action: func(c *cli.Context) error {
						if !utils.AskForConfirmation("You will lose existing changes.") {
							os.Exit(1)
						}
						return MustInitWorkflow(cfg, manifest).MassDone(c.Bool(noOperation))
					},
				},
				{
					Name:    "update",
					Aliases: []string{"u"},
					Usage:   "Clean the repository, checkout master and pull.",
					Action: func(c *cli.Context) error {
						if !utils.AskForConfirmation("You will lose existing changes.") {
							os.Exit(1)
						}
						return MustInitWorkflow(cfg, manifest).MassUpdate()
					},
				},
			},
		},
	}
}

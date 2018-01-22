package cmd

import (
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/atlassian"
	"github.com/benchlabs/bub/utils"
	"github.com/urfave/cli"
	"log"
	"strings"
)

const jiraBeeDesc = "Must use the browser event if Bee is present."

func buildJIRACmds(cfg *core.Configuration) []cli.Command {
	reactive := "reactive"
	project := "project"
	transition := "transition"

	return []cli.Command{
		buildJIRAOpenBoardCmd(cfg),
		buildJIRASearchIssueCmd(cfg),
		buildJIRAOpenRecentlyAccessedIssuesCmd(cfg),
		buildJIRAClaimIssueCmd(cfg),
		{
			Name:      "create",
			Aliases:   []string{"c"},
			Usage:     "Creates a JIRA issue.",
			ArgsUsage: "SUMMARY DESCRIPTION ... [ARGS]",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: reactive, Usage: "The issue will be added to the current sprint."},
				cli.StringFlag{Name: project, Usage: "Sets project, uses the default project is not set."},
				cli.StringFlag{Name: transition, Usage: "Set the issue transition. e.g. Done."},
			},
			Action: func(c *cli.Context) error {
				if len(c.Args()) < 2 {
					log.Fatal("The summary (title) and description must be passed.")
				}
				summary := c.Args().Get(0)
				desc := c.Args().Get(1)
				return atlassian.MustInitJIRA(cfg).CreateIssue(c.String(project), summary, desc, c.String(transition), c.Bool(reactive))
			},
		},
		buildJIRAOpenIssueCmd(cfg),
		buildJIRAViewIssueCmd(cfg),
		buildJIRACommentOnIssuesCmd(cfg),
		buildJIRAListAssignedIssuesCmd(cfg),
		buildJIRATransitionIssueCmd(cfg),
	}
}

func buildJIRASearchIssueCmd(cfg *core.Configuration) cli.Command {
	all := "all"
	resolved := "resolved"
	project := "p"
	browse := "b"
	jql := "jql"

	return cli.Command{
		Name:    "search",
		Aliases: []string{"s"},
		Usage:   "Search and open JIRA issue in the browser.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: all, Usage: "Use all projects."},
			cli.BoolFlag{Name: resolved, Usage: "Include resolved issues."},
			cli.StringFlag{Name: project, Usage: "Specify the project."},
			cli.BoolFlag{Name: browse, Usage: jiraBeeDesc},
			cli.BoolFlag{Name: jql, Usage: "Query with JQL instead of text search."},
		},
		Action: func(c *cli.Context) error {
			project := c.String("pr")
			if !c.Bool(all) {
				project = cfg.JIRA.Project
			}
			query := strings.Join(c.Args(), " ")
			if c.Bool(jql) {
				return atlassian.MustInitJIRA(cfg).SearchIssueJQL(query, c.Bool(browse))
			}
			return atlassian.MustInitJIRA(cfg).SearchIssueText(
				query,
				project,
				c.Bool(resolved),
				c.Bool(browse))
		},
	}
}

func buildJIRAOpenRecentlyAccessedIssuesCmd(cfg *core.Configuration) cli.Command {
	browse := "b"

	return cli.Command{
		Name:    "recent",
		Aliases: []string{"r"},
		Usage:   "Pick and open issue that you recently interacted with.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: browse, Usage: jiraBeeDesc},
		},
		Action: func(c *cli.Context) error {
			return atlassian.MustInitJIRA(cfg).OpenRecentlyAccessedIssues(c.Bool(browse))
		},
	}
}

func buildJIRAOpenIssueCmd(cfg *core.Configuration) cli.Command {
	browse := "b"

	return cli.Command{
		Name:    "open",
		Aliases: []string{"o"},
		Usage:   "Open JIRA issue in the browser.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: browse, Usage: jiraBeeDesc},
		},
		Action: func(c *cli.Context) error {
			var key string
			if len(c.Args()) > 0 {
				key = c.Args().Get(0)
			}
			return atlassian.MustInitJIRA(cfg).OpenIssue(key, c.Bool(browse))
		},
	}
}

func buildJIRAViewIssueCmd(cfg *core.Configuration) cli.Command {
	return cli.Command{
		Name:    "view",
		Aliases: []string{"v"},
		Usage:   "View JIRA issue in the terminal.",
		Action: func(c *cli.Context) error {
			var key string
			if len(c.Args()) > 0 {
				key = c.Args().Get(0)
			}
			return atlassian.MustInitJIRA(cfg).ViewIssue(key)
		},
	}
}

func buildJIRAClaimIssueCmd(cfg *core.Configuration) cli.Command {
	return cli.Command{
		Name:    "claim",
		Aliases: []string{"cl"},
		Usage:   "Claim unassigned issue in the current sprint.",
		Action: func(c *cli.Context) error {
			var issueKey string
			if len(c.Args()) > 0 {
				issueKey = c.Args().Get(0)
			}
			return atlassian.MustInitJIRA(cfg).ClaimIssueInActiveSprint(issueKey)
		},
	}
}

func buildJIRAListAssignedIssuesCmd(cfg *core.Configuration) cli.Command {
	showDescription := "d"
	return cli.Command{
		Name:    "assigned",
		Aliases: []string{"a"},
		Usage:   "Show assigned issues.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: showDescription, Usage: jiraBeeDesc},
		},
		Action: func(c *cli.Context) error {
			return atlassian.MustInitJIRA(cfg).ListAssignedIssues(c.Bool(showDescription))
		},
	}
}

func buildJIRACommentOnIssuesCmd(cfg *core.Configuration) cli.Command {
	return cli.Command{
		Name:    "comment",
		Aliases: []string{"co"},
		Usage:   "COMMENT [ISSUE-KEY] Add comment to issue.",
		Action: func(c *cli.Context) error {
			key := ""
			if len(c.Args()) > 1 {
				key = c.Args()[1]
			}
			return atlassian.MustInitJIRA(cfg).CommentOnIssue(key, c.Args().First())
		},
	}
}

func buildJIRATransitionIssueCmd(cfg *core.Configuration) cli.Command {
	return cli.Command{
		Name:    "transition",
		Aliases: []string{"t"},
		Usage:   "Transition issue based on current branch.",
		Action: func(c *cli.Context) error {
			var transition string
			if len(c.Args()) > 0 {
				transition = c.Args().Get(0)
			}
			return atlassian.MustInitJIRA(cfg).TransitionIssue("", transition)
		},
	}
}

func buildJIRAOpenBoardCmd(cfg *core.Configuration) cli.Command {
	return cli.Command{
		Name:    "board",
		Aliases: []string{"b"},
		Usage:   "Open your JIRA board.",
		Action: func(c *cli.Context) error {
			return utils.OpenURI(cfg.JIRA.Server, "secure/RapidBoard.jspa?rapidView="+cfg.JIRA.Board)
		},
	}
}

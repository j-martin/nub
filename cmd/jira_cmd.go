package cmd

import (
	"github.com/j-martin/bub/core"
	"github.com/j-martin/bub/integrations/atlassian"
	"github.com/j-martin/bub/utils"
	"github.com/urfave/cli"
	"log"
	"strings"
)

const jiraBeeDesc = "Open with Bee, if present."

func buildJIRACmds(cfg *core.Configuration) []cli.Command {
	return []cli.Command{
		buildJIRAOpenBoardCmd(cfg),
		buildJIRASearchIssueCmd(cfg),
		buildJIRAOpenRecentlyAccessedIssuesCmd(cfg),
		buildJIRAClaimIssueCmd(cfg),
		buildJIRACreateIssueCmd(cfg),
		buildJIRAOpenIssueCmd(cfg),
		buildJIRAViewIssueCmd(cfg),
		buildJIRACommentOnIssuesCmd(cfg),
		buildJIRAListAssignedIssuesCmd(cfg),
		buildJIRATransitionIssueCmd(cfg),
		buildJIRListWorkDayCmd(cfg),
	}
}

func buildJIRACreateIssueCmd(cfg *core.Configuration) cli.Command {
	reactive := "reactive"
	project := "project"
	transition := "transition"

	return cli.Command{
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
	}
}

func buildJIRASearchIssueCmd(cfg *core.Configuration) cli.Command {
	all := "all"
	resolved := "resolved"
	project := "p"
	bee := "b"
	jql := "jql"

	return cli.Command{
		Name:    "search",
		Aliases: []string{"s"},
		Usage:   "Search and open JIRA issue in the browser.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: all, Usage: "Use all projects."},
			cli.BoolFlag{Name: resolved, Usage: "Include resolved issues."},
			cli.StringFlag{Name: project, Usage: "Specify the project."},
			cli.BoolFlag{Name: bee, Usage: jiraBeeDesc},
			cli.BoolFlag{Name: jql, Usage: "Query with JQL instead of text search."},
		},
		Action: func(c *cli.Context) error {
			project := c.String("pr")
			if !c.Bool(all) {
				project = cfg.JIRA.Project
			}
			query := strings.Join(c.Args(), " ")
			if c.Bool(jql) {
				return atlassian.MustInitJIRA(cfg).SearchIssueJQL(query, c.Bool(bee))
			}
			return atlassian.MustInitJIRA(cfg).SearchIssueText(
				query,
				project,
				c.Bool(resolved),
				c.Bool(bee))
		},
	}
}

func buildJIRAOpenRecentlyAccessedIssuesCmd(cfg *core.Configuration) cli.Command {
	bee := "b"

	return cli.Command{
		Name:    "recent",
		Aliases: []string{"r"},
		Usage:   "Pick and open issue that you recently interacted with.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: bee, Usage: jiraBeeDesc},
		},
		Action: func(c *cli.Context) error {
			return atlassian.MustInitJIRA(cfg).OpenRecentlyAccessedIssues(c.Bool(bee))
		},
	}
}

func buildJIRAOpenIssueCmd(cfg *core.Configuration) cli.Command {
	bee := "b"

	return cli.Command{
		Name:    "open",
		Aliases: []string{"o"},
		Usage:   "Open JIRA issue in the browser.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: bee, Usage: jiraBeeDesc},
		},
		Action: func(c *cli.Context) error {
			var key string
			if len(c.Args()) > 0 {
				key = c.Args().Get(0)
			}
			return atlassian.MustInitJIRA(cfg).OpenIssue(key, c.Bool(bee))
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
		Aliases: []string{"t", "tr"},
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
		Aliases: []string{"bo"},
		Usage:   "Open your JIRA board.",
		Action: func(c *cli.Context) error {
			return utils.OpenURI(cfg.JIRA.Server, "secure/RapidBoard.jspa?rapidView="+cfg.JIRA.Board)
		},
	}
}
func buildJIRListWorkDayCmd(cfg *core.Configuration) cli.Command {
	prefix := "prefix"
	orgFormat := "org"
	return cli.Command{
		Name:    "workday",
		Aliases: []string{"d"},
		Usage:   "List the work during a specific day.",
		Flags: []cli.Flag{
			cli.StringFlag{Name: prefix, Usage: "String to prefix the issue."},
			cli.BoolFlag{Name: orgFormat, Usage: "Format for org-mode."},
		},
		Action: func(c *cli.Context) error {
			var date string
			if len(c.Args()) > 0 {
				date = c.Args().Get(0)
			}
			return atlassian.MustInitJIRA(cfg).ListWorkDay(date, c.String(prefix), c.Bool(orgFormat))
		},
	}
}

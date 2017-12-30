package main

import (
	"fmt"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations"
	"github.com/benchlabs/bub/utils"
	"log"
	"github.com/benchlabs/bub/integrations/atlassian"
)

type Workflow struct {
	cfg      *core.Configuration
	git      *core.Git
	github   *integrations.GitHub
	jira     *atlassian.JIRA
	manifest *core.Manifest
}

func MustInitWorkflow(cfg *core.Configuration, manifest *core.Manifest) *Workflow {
	return &Workflow{
		cfg:      cfg,
		git:      core.MustInitGit(),
		github:   integrations.MustInitGitHub(cfg),
		jira:     atlassian.MustInitJIRA(cfg),
		manifest: manifest,
	}
}

func (wf *Workflow) Git() *core.Git {
	if wf.git == nil {
		wf.git = core.MustInitGit()
	}
	return wf.git
}

func (wf *Workflow) GitHub() *integrations.GitHub {
	if wf.github == nil {
		wf.github = integrations.MustInitGitHub(wf.cfg)
	}
	return wf.github
}

func (wf *Workflow) JIRA() *atlassian.JIRA {
	if wf.jira == nil {
		wf.jira = atlassian.MustInitJIRA(wf.cfg)
	}
	return wf.jira
}

func (wf *Workflow) MassUpdate() error {
	return core.ForEachRepo(func() error {
		wf.Git().CleanAndUpdate()
		return nil
	})
}

func (wf *Workflow) MassStart() error {
	issue, err := wf.JIRA().PickAssignedIssue()
	if err != nil {
		return err
	}

	return core.ForEachRepo(func() error {
		wf.Git().CleanAndUpdate()
		return wf.JIRA().CreateBranchFromIssue(issue)
	})
}

func (wf *Workflow) MassDone(noop bool) error {
	return core.ForEachRepo(func() error {
		g := core.MustInitGit()
		if g.ContainedUncommittedChanges() {
			utils.ConditionalOp("Committing.", noop, func() error {
				g.CommitWithBranchName()
				return nil
			})
		}

		if !g.IsDifferentFromMaster() {
			log.Printf("No commits. Skipping.")
			return nil
		}

		utils.ConditionalOp("Pushing", noop, func() error {
			g.Push(wf.cfg)
			return nil
		})
		return nil
	})
}

func (wf *Workflow) CreatePR(title, body string) error {
	err := wf.GitHub().CreatePR(title, body)
	if err != nil {
		return err
	}
	return wf.JIRA().TransitionIssue("", "review")
}

func (wf *Workflow) Log() error {
	c, err := wf.Git().PickCommit(wf.Git().Log())
	if err != nil {
		return err
	}
	return wf.OpenCommit(c)
}

func (wf *Workflow) OpenCommit(c *core.GitCommit) error {
	issueKey := wf.Git().GetIssueRegex().FindString(c.Subject)
	pr := wf.Git().GetPRRegex().FindStringSubmatch(c.Subject)

	openList := make(map[string]func() error)
	if len(pr) > 2 && pr[2] != "" {
		openList["GitHub PR"] = func() error {
			return wf.GitHub().OpenPR(wf.manifest, pr[2])
		}
	}
	if issueKey != "" {
		openList["JIRA"] = func() error {
			return wf.JIRA().OpenIssueFromKey(issueKey)
		}
	}
	if len(openList) > 0 {
		var titles []string
		for k := range openList {
			titles = append(titles, k)
		}
		result, err := utils.PickItem("Open", titles)
		if err != nil {
			return err
		}
		return openList[result]()
	}
	fmt.Printf("%v", c)
	return nil
}

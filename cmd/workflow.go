package cmd

import (
	"fmt"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations"
	"github.com/benchlabs/bub/integrations/atlassian"
	"github.com/benchlabs/bub/utils"
	"log"
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
		git:      core.InitGit(),
		github:   integrations.MustInitGitHub(cfg),
		jira:     atlassian.MustInitJIRA(cfg),
		manifest: manifest,
	}
}

func (wf *Workflow) Git() *core.Git {
	if wf.git == nil {
		wf.git = core.InitGit()
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
	return core.ForEachRepo(func(repoDir string) error {
		return core.MustInitGit(repoDir).Sync(true)
	})
}

func (wf *Workflow) MassStart() error {
	issue, err := wf.JIRA().PickAssignedIssue()
	if err != nil {
		return err
	}

	return core.ForEachRepo(func(repo string) error {
		g := core.MustInitGit(repo)
		err := g.Sync(true)
		if err != nil {
			return err
		}
		return wf.JIRA().CreateBranchFromIssue(repo, issue)
	})
}

func (wf *Workflow) MassDone(noop bool) error {
	return core.ForEachRepo(func(repoDir string) error {
		g := core.MustInitGit(repoDir)
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
			wf.GitHub().CreatePR("", "", repoDir)
			return nil
		})
		return nil
	})
}

func (wf *Workflow) CreatePR(title, body string, review bool) error {
	err := wf.GitHub().CreatePR(title, body, "")
	if err != nil {
		return err
	}
	if review {
		return wf.JIRA().TransitionIssue("", "review")
	}
	return nil
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

	openList := map[string]func() error{
		"GitHub Commit": func() error {
			return wf.GitHub().OpenCommit(wf.manifest, c)
		},
		"GitHub Compare with Master": func() error {
			return wf.GitHub().OpenCompare(wf.manifest, c, "master")
		},
	}
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

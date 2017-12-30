package main

import (
	"fmt"
	"log"
)

type Workflow struct {
	cfg      *Configuration
	git      *Git
	github   *GitHub
	jira     *JIRA
	manifest *Manifest
}

func MustInitWorkflow(cfg *Configuration, manifest *Manifest) *Workflow {
	return &Workflow{
		cfg:      cfg,
		git:      MustInitGit(),
		github:   MustInitGitHub(cfg),
		jira:     MustInitJIRA(cfg),
		manifest: manifest,
	}
}

func (wf *Workflow) Git() *Git {
	if wf.git == nil {
		wf.git = MustInitGit()
	}
	return wf.git
}

func (wf *Workflow) GitHub() *GitHub {
	if wf.github == nil {
		wf.github = MustInitGitHub(wf.cfg)
	}
	return wf.github
}

func (wf *Workflow) JIRA() *JIRA {
	if wf.jira == nil {
		wf.jira = MustInitJIRA(wf.cfg)
	}
	return wf.jira
}

func (wf *Workflow) MassUpdate() error {
	return ForEachRepo(func() error {
		wf.Git().Update()
		return nil
	})
}

func (wf *Workflow) MassStart() error {
	issue, err := wf.JIRA().PickAssignedIssue()
	if err != nil {
		return err
	}

	return ForEachRepo(func() error {
		wf.Git().Update()
		return wf.JIRA().CreateBranchFromIssue(issue)
	})
}

func (wf *Workflow) MassDone(noop bool) error {
	return ForEachRepo(func() error {
		g := MustInitGit()
		if g.ContainedUncommittedChanges() {
			ConditionalOp("Committing.", noop, func() error {
				MustRunCmd("git", "commit", "-m", g.GetTitleFromBranchName(), "--all")
				return nil
			})
		}

		if !g.IsDifferentFromMaster() {
			log.Printf("No commits. Skipping.")
			return nil
		}

		ConditionalOp("Pushing", noop, func() error {
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

func (wf *Workflow) OpenCommit(c *GitCommit) error {
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
		result, err := PickItem("Open", titles)
		if err != nil {
			return err
		}
		return openList[result]()
	}
	fmt.Printf("%v", c)
	return nil
}

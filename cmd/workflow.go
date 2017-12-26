package main

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"regexp"
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

func (w *Workflow) Log() error {
	cs, err := w.git.repository.Log(&git.LogOptions{})
	if err != nil {
		return err
	}
	var commits []*object.Commit
	i := 0
	for i < 100 {
		c, err := cs.Next()
		if err != nil && err.Error() == "EOF" {
			break
		} else if err != nil {
			return err
		}
		commits = append(commits, c)
		i++
	}
	c, err := w.git.pickCommit(commits)
	if err != nil {
		return err
	}
	return w.OpenCommit(c)
}

func (w *Workflow) OpenCommit(c *object.Commit) error {
	issueRegex := regexp.MustCompile("([A-Z]{2,}-\\d+)")
	issueKey := issueRegex.FindString(c.Message)
	issueRegex = regexp.MustCompile("(Merge pull request #)(\\d+) from \\w+/")
	pr := issueRegex.FindStringSubmatch(c.Message)

	openList := make(map[string]func() error)
	if len(pr) > 2 && pr[2] != "" {
		openList["GitHub PR"] = func() error {
			return w.github.OpenPR(w.manifest, pr[2])
		}
	}
	if issueKey != "" {
		openList["JIRA"] = func() error {
			return w.jira.OpenIssueFromKey(issueKey)
		}
	}
	if len(openList) > 0 {
		var titles []string
		for k := range openList {
			titles = append(titles, k)
		}
		result, err := pickItem("Open", titles)
		if err != nil {
			return err
		}
		return openList[result]()
	}
	fmt.Printf("%v", c)
	return nil
}

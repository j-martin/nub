package main

import (
	"github.com/andygrunwald/go-jira"
	"github.com/manifoldco/promptui"
	"log"
	"strings"
)

type JIRA struct {
	client *jira.Client
	cfg    *Configuration
}

func MustInitJIRA(cfg *Configuration) *JIRA {
	j := JIRA{}
	loadCredentials("JIRA", &cfg.JIRA)
	if err := j.init(cfg); err != nil {
		log.Fatalf("Failed to initiate JIRA client: %v", err)
	}
	return &j
}

func (j *JIRA) init(cfg *Configuration) error {
	checkServerConfig(cfg.JIRA)
	client, err := jira.NewClient(nil, cfg.JIRA.Server)
	if err != nil {
		return err
	}
	client.Authentication.SetBasicAuth(cfg.JIRA.Username, cfg.JIRA.Password)

	j.client = client
	j.cfg = cfg
	return err
}

func (j *JIRA) getAssignedIssues() ([]jira.Issue, error) {
	jql := "resolution = null AND assignee=currentUser() ORDER BY Rank"
	issues, _, err := j.client.Issue.Search(jql, &jira.SearchOptions{MaxResults: 50})
	return issues, err
}

func (j *JIRA) CreateBranchFromAssignedIssues() error {
	issues, err := j.getAssignedIssues()
	if err != nil {
		return err
	}
	issue, err := pickIssue(issues)
	if err != nil {
		return err
	}
	CreateBranch(issue.Key + " " + issue.Fields.Summary)
	return nil
}

func (j *JIRA) OpenJIRAIssue() error {
	key := GetIssueKeyFromBranch()
	if key == "" {
		is, err := j.getAssignedIssues()
		if err != nil {
			return err
		}
		i, err := pickIssue(is)
		if err != nil {
			return err
		}
		key = i.Key
	}
	beeInstalled, err := pathExists("/Applications/Bee.app")
	if err != nil {
		return nil
	}
	if beeInstalled {
		openURI("bee://item?id=" + key)
		return nil
	}
	openURI(j.cfg.JIRA.Server, "browse", key)
	return nil
}

func pickIssue(issues []jira.Issue) (jira.Issue, error) {
	templates := &promptui.SelectTemplates{
		Label: "{{ . }}:",
		Active: "▶ {{ .Key }}	{{ .Fields.Summary }}",
		Inactive: "  {{ .Key }}	{{ .Fields.Summary }}",
		Selected: "▶ {{ .Key }}	{{ .Fields.Summary }}",
		Details: `
--------- Issue ----------
{{ "Key:" | faint }}	{{ .Key }}
{{ "Summary:" | faint }}	{{ .Fields.Summary }}
{{ "Description:" | faint }}	{{ .Fields.Description }}
`,
	}

	searcher := func(input string, index int) bool {
		i := issues[index]
		name := strings.Replace(strings.ToLower(i.Fields.Summary), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Size:      20,
		Label:     "Pick an issue",
		Items:     issues,
		Templates: templates,
		Searcher:  searcher,
	}
	i, _, err := prompt.Run()
	return issues[i], err
}

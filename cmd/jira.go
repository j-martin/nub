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
	issue, err := j.pickIssue(issues)
	if err != nil {
		return err
	}
	CreateBranch(issue.Key + " " + issue.Fields.Summary)
	return nil
}

func (j *JIRA) sanitizeTransitionName(tr string) string {
	r := strings.NewReplacer("'", "", " ", "")
	return r.Replace(strings.ToLower(tr))
}

func (j *JIRA) matchTransition(key, transitionName string) (jira.Transition, error) {
	trs, _, err := j.client.Issue.GetTransitions(key)
	if err != nil {
		return jira.Transition{}, err
	}
	for _, tr := range trs {
		if j.sanitizeTransitionName(tr.Name) == j.sanitizeTransitionName(transitionName) {
			return tr, nil
		}
	}

	return j.pickTransition(trs)
}

func (j *JIRA) TransitionIssue(transitionName string) error {
	key, err := j.GetIssueKeyFromBranchOrAssigned()
	if err != nil {
		return err
	}
	transition, err := j.matchTransition(key, transitionName)
	if err != nil {
		return err
	}
	_, err = j.client.Issue.DoTransition(key, transition.ID)
	if err != nil {
		return err
	}

	log.Printf("Issue %v transitoned to %v", key, transition.Name)
	return nil
}

func (j *JIRA) GetIssueKeyFromBranchOrAssigned() (string, error) {
	key := GetIssueKeyFromBranch()
	if key == "" {
		log.Print("No issue key found in ")
		is, err := j.getAssignedIssues()
		if err != nil {
			return "", err
		}
		i, err := j.pickIssue(is)
		if err != nil {
			return "", err
		}
		key = i.Key
	}
	return key, nil
}

func (j *JIRA) OpenIssue() error {
	key, err := j.GetIssueKeyFromBranchOrAssigned()
	if err != nil {
		return nil
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

func (j *JIRA) pickIssue(issues []jira.Issue) (jira.Issue, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "▶ {{ .Key }}	{{ .Fields.Summary }}",
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

func (j *JIRA) pickTransition(transitions []jira.Transition) (jira.Transition, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "▶ {{ .Name }}	{{ .ID }}",
		Inactive: "  {{ .Name }}	{{ .ID }}",
		Selected: "▶ {{ .Name }}	{{ .ID }}",
	}

	searcher := func(input string, index int) bool {
		i := transitions[index]
		name := strings.Replace(strings.ToLower(i.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Size:      20,
		Label:     "Pick an transition",
		Items:     transitions,
		Templates: templates,
		Searcher:  searcher,
	}
	i, _, err := prompt.Run()
	return transitions[i], err
}

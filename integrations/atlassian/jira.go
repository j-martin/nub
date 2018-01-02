package atlassian

import (
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/utils"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"strings"
)

type JIRA struct {
	client *jira.Client
	cfg    *core.Configuration
}

func MustInitJIRA(cfg *core.Configuration) *JIRA {
	j := JIRA{}
	core.LoadCredentials("JIRA", &cfg.JIRA.Username, &cfg.JIRA.Password)
	if err := j.init(cfg); err != nil {
		log.Fatalf("Failed to initiate JIRA client: %v", err)
	}
	return &j
}

func (j *JIRA) init(cfg *core.Configuration) error {
	core.CheckServerConfig(cfg.JIRA.Server)
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
	return j.search("resolution = null AND assignee=currentUser() ORDER BY Rank")
}

func (j *JIRA) SearchIssueText(text, project string, resolved bool) error {
	is, err := j.SearchText(text, project, resolved)
	if err != nil {
		return err
	}
	i, err := j.pickIssue(is)
	if err != nil {
		return err
	}
	return j.openIssue(i)
}
func (j *JIRA) SearchText(text, project string, resolved bool) ([]jira.Issue, error) {
	jql := fmt.Sprintf("text ~ \"%v\" ORDER BY createdDate", text)
	if project != "" {
		jql = fmt.Sprintf("project = %v AND %v", project, jql)
	}
	if !resolved {
		jql = fmt.Sprintf("resolution = null AND %v", jql)
	}
	return j.search(jql)
}

func (j *JIRA) getUnassignedIssuesInSprint() ([]jira.Issue, error) {
	jql := fmt.Sprintf("project = %v AND sprint IN openSprints() AND assignee = null AND resolution = null ORDER BY Rank", j.cfg.JIRA.Project)
	return j.search(jql)
}

func (j *JIRA) search(jql string) ([]jira.Issue, error) {
	issues, _, err := j.client.Issue.Search(jql, &jira.SearchOptions{MaxResults: 50})
	return issues, err
}

func (j *JIRA) ClaimIssueInActiveSprint() error {
	is, err := j.getUnassignedIssuesInSprint()
	if err != nil {
		return err
	}
	i, err := j.pickIssue(is)
	if err != nil {
		return err
	}
	i.Fields.Assignee = &jira.User{Name: j.cfg.JIRA.Username}
	_, res, err := j.client.Issue.Update(&i)
	if err != nil {
		b, _ := ioutil.ReadAll(res.Body)
		log.Print(string(b))
		return err
	}
	err = j.TransitionIssue(i.Key, "inprogress")
	if err != nil {
		return err
	}
	log.Printf("%v claimed.", i.Key)
	return nil
}

func (j *JIRA) PickAssignedIssue() (jira.Issue, error) {
	issues, err := j.getAssignedIssues()
	if err != nil {
		return jira.Issue{}, err
	}
	return j.pickIssue(issues)
}

func (j *JIRA) CreateBranchFromAssignedIssue() error {
	issue, err := j.PickAssignedIssue()
	if err != nil {
		return err
	}
	return j.CreateBranchFromIssue("",issue)
}

func (j *JIRA) CreateBranchFromIssue(repoDir string,issue jira.Issue) error {
	core.MustInitGit(repoDir).CreateBranch(issue.Key + " " + issue.Fields.Summary)
	return nil
}

func (j *JIRA) sanitizeTransitionName(tr string) string {
	r := strings.NewReplacer("'", "", " ", "", "in", "")
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

func (j *JIRA) TransitionIssue(key, transitionName string) (err error) {
	if key == "" {
		key, err = j.getIssueKeyFromBranchOrAssigned()
		if err != nil {
			return err
		}
	}
	transition, err := j.matchTransition(key, transitionName)
	if err != nil {
		return err
	}
	_, err = j.client.Issue.DoTransition(key, transition.ID)
	if err != nil {
		return err
	}

	log.Printf("%v transitoned to %v", key, transition.Name)
	return nil
}

func (j *JIRA) getIssueKeyFromBranchOrAssigned() (string, error) {
	key := core.InitGit().GetIssueKeyFromBranch()
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

func (j JIRA) CreateIssue(project, summary, description, transition string, reactive bool) error {
	if project == "" && j.cfg.JIRA.Project != "" {
		project = j.cfg.JIRA.Project
	} else if project == "" {
		return errors.New("the project must be defined (in the argument or the config)")
	}
	fields := jira.IssueFields{
		Summary:     summary,
		Description: description,
		Project:     jira.Project{Key: project},
		Type: jira.IssueType{
			Name: "Task",
		},
	}

	if reactive {
		fields.Assignee = &jira.User{Name: j.cfg.JIRA.Username}
	}

	i, res, err := j.client.Issue.Create(&jira.Issue{Fields: &fields})
	if err != nil {
		b, _ := ioutil.ReadAll(res.Body)
		log.Print(string(b))
		return err
	}
	log.Printf("%v created.", i.Key)
	if transition != "" {
		if err = j.TransitionIssue(i.Key, transition); err != nil {
			return err
		}
	}
	if reactive {
		sp, err := j.getActiveSprint()
		if err != nil {
			return err
		}
		j.client.Sprint.MoveIssuesToSprint(sp.ID, []string{i.Key})
		log.Printf("%v moved to the active sprint.", i.Key)
	}
	return nil
}

func (j *JIRA) getActiveSprint() (jira.Sprint, error) {
	empty := jira.Sprint{}
	if j.cfg.JIRA.Board == "" {
		return empty, errors.New("the board id must be defined in the config")
	}
	sps, _, err := j.client.Board.GetAllSprints(j.cfg.JIRA.Board)
	if err != nil {
		return empty, err
	}
	for _, sp := range sps {
		if sp.State == "active" {
			return sp, nil
		}
	}

	return empty, errors.New("no active sprint found")
}

func (j *JIRA) openIssue(issue jira.Issue) error {
	return j.OpenIssueFromKey(issue.Key)
}

func (j *JIRA) OpenIssueFromKey(key string) error {
	beeInstalled, err := utils.PathExists("/Applications/Bee.app")
	if err != nil {
		return nil
	}
	if beeInstalled {
		utils.OpenURI("bee://item?id=" + key)
		return nil
	}
	utils.OpenURI(j.cfg.JIRA.Server, "browse", key)
	return nil
}

func (j *JIRA) OpenIssue() error {
	key, err := j.getIssueKeyFromBranchOrAssigned()
	if err != nil {
		return nil
	}
	return j.OpenIssueFromKey(key)
}

func (j *JIRA) pickIssue(issues []jira.Issue) (jira.Issue, error) {
	if len(issues) == 0 {
		return jira.Issue{}, errors.New("no issue to pick")
	}
	if len(issues) == 1 {
		issue := issues[0]
		log.Printf("%v %v only available", issue.Key, issue.Fields.Summary)
		return issue, nil
	}
	templates := &promptui.SelectTemplates{
		Label: "{{ . }}:",
		Active: "▶ {{ .Key }}	{{ .Fields.Summary }}",
		Inactive: "  {{ .Key }}	{{ .Fields.Summary }}",
		Selected: "▶ {{ .Key }}	{{ .Fields.Summary }}",
		Details: `
--------- Issue ----------
{{ "Key:" | faint }}	{{ .Key }}
{{ "Assignee:" | faint }}	{{ if .Fields.Assignee }}{{ .Fields.Assignee.DisplayName }}{{ end }}
{{ "Reporter:" | faint }}	{{ .Fields.Reporter.DisplayName }}
{{ "Status:" | faint }}	{{ .Fields.Status.Name }}
{{ "Summary:" | faint }}	{{ .Fields.Summary }}
{{ "Description:" | faint }}	{{ .Fields.Description }}
`,
	}

	searcher := func(input string, index int) bool {
		i := issues[index]
		name := strings.Replace(strings.ToLower(i.Key+i.Fields.Summary), " ", "", -1)
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
	if len(transitions) == 0 {
		return jira.Transition{}, errors.New("no transition to pick")
	}
	if len(transitions) == 1 {
		return transitions[0], nil
	}
	templates := &promptui.SelectTemplates{
		Label: "{{ . }}:",
		Active: "▶ {{ .Name }}	{{ .ID }}",
		Inactive: "  {{ .Name }}	{{ .ID }}",
		Selected: "▶ {{ .Name }}	{{ .ID }}",
		Details: "",
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

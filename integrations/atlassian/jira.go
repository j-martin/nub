package atlassian

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/j-martin/nub/core"
	"github.com/j-martin/nub/utils"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/trivago/tgo/tcontainer"
)

type JIRA struct {
	client *jira.Client
	cfg    *core.Configuration
}

func MustInitJIRA(cfg *core.Configuration) *JIRA {
	j := JIRA{}
	mustLoadJIRACredentials(cfg)
	err := j.init(cfg)
	if err != nil {
		log.Fatalf("Failed to initiate JIRA client: %v", err)
	}
	return &j
}

func mustLoadJIRACredentials(cfg *core.Configuration) {
	err := core.LoadCredentials("JIRA", &cfg.JIRA.Username, &cfg.JIRA.Password, cfg.ResetCredentials)
	if err != nil {
		log.Fatalf("Failed to set JIRA credentials: %v", err)
	}
}

func MustSetupJIRA(cfg *core.Configuration) {
	if utils.AskForConfirmation(
		"Enter your Atlassian credentials. Refer to your profile page to see your username. " +
			"You may have to ask to reset your password if you never used GSuite or Okta to login to JIRA.\n" +
			"Open the profile page?") {
		utils.OpenURI(cfg.JIRA.Server, "secure/ViewProfile.jspa")
	}
	mustLoadJIRACredentials(cfg)
}

func (j *JIRA) IsEnabled() bool {
	return j.cfg.JIRA.Enabled
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

func (j *JIRA) SearchIssueJQL(jql string, useBee bool) error {
	is, err := j.search(jql)
	if err != nil {
		return err
	}
	return j.pickAndOpenIssue(is, useBee)
}

func (j *JIRA) SearchIssueText(text, project string, resolved, useBee bool) error {
	is, err := j.SearchText(text, project, resolved)
	if err != nil {
		return err
	}
	return j.pickAndOpenIssue(is, useBee)
}

func (j *JIRA) pickAndOpenIssue(is []jira.Issue, useBee bool) error {
	i, err := j.pickIssue(is)
	if err != nil {
		return err
	}
	return j.openIssue(i, useBee)
}

func (j *JIRA) OpenRecentlyAccessedIssues(useBee bool) error {
	jql := "assignee was currentUser() or reporter was currentUser() or issuekey in issueHistory() order by updatedDate"
	return j.SearchIssueJQL(jql, useBee)
}

func (j *JIRA) ListWorkDay(date, prefix string, orgFormat bool) error {
	now := time.Now().Format("2006-01-02")
	if date == "" {
		date = now
	}
	log.Printf("Listing work done on: %s", date)
	jql := fmt.Sprintf(`assignee = currentUser() AND ((status WAS IN ("In Progress", Review) ON "%s" and (resolutiondate >= "%s" OR resolution IS NULL)) OR (resolution CHANGED ON "%s")) ORDER BY key`, date, date, date)
	log.Printf("Query: %s", jql)
	is, err := j.search(jql)
	if err != nil {
		return err
	}

	var format string
	if orgFormat {
		format = `{{.Prefix}}[[{{.Server}}/browse/{{.Issue.Key}}][{{.Issue.Key}}]] {{.Issue.Fields.Summary}}
Status: {{.Issue.Fields.Status.Name}} (as of {{.Now}})
`
	} else {
		format = `{{.Prefix}}{{.Issue.Key}} {{.Issue.Fields.Summary}}
`
	}
	tmp, err := template.New("issue").Parse(format)
	for _, i := range is {
		a := struct {
			Prefix, Server, Now string
			Issue               jira.Issue
		}{
			Prefix: prefix,
			Server: j.cfg.JIRA.Server,
			Now:    now,
			Issue:  i,
		}
		tmp.Execute(os.Stdout, a)
	}
	return nil
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
	jql := fmt.Sprintf("project = %v  AND assignee = null AND resolution = null ORDER BY Rank", j.cfg.JIRA.Project)
	return j.search(jql)
}

func (j *JIRA) search(jql string) ([]jira.Issue, error) {
	issues, _, err := j.client.Issue.Search(jql, &jira.SearchOptions{MaxResults: 50})
	return issues, err
}

func (j *JIRA) ClaimIssueInActiveSprint(key string) error {
	if key != "" {
		i, _, err := j.client.Issue.Get(key, &jira.GetQueryOptions{})
		if err != nil {
			return err
		}
		return j.claimIssue(i)
	}
	is, err := j.getUnassignedIssuesInSprint()
	if err != nil {
		return err
	}
	i, err := j.pickIssue(is)
	if err != nil {
		return err
	}
	return j.claimIssue(i)
}

func (j *JIRA) claimIssue(i *jira.Issue) error {
	key := i.Key
	updatedIssue := &jira.Issue{
		Key: key,
		Fields: &jira.IssueFields{
			Type:        i.Fields.Type,
			Summary:     i.Fields.Summary,
			Description: i.Fields.Description,
			Assignee:    &jira.User{Name: j.cfg.JIRA.Username},
		},
	}
	_, res, err := j.client.Issue.Update(updatedIssue)
	if err != nil {
		j.logBody(res)
		return err
	}
	err = j.TransitionIssue(key, "progress")
	if err != nil {
		return err
	}
	log.Printf("%v claimed.", key)
	err = j.MoveIssueToCurrentSprint(i)
	if err != nil {
		return err
	}
	if utils.IsRepository(".") && utils.AskForConfirmation("Create the branch for this issue?") {
		j.CreateBranchFromIssue(i, ".", false)
	}
	return nil
}

func (j *JIRA) logBody(res *jira.Response) {
	b, _ := ioutil.ReadAll(res.Body)
	log.Print(string(b))
}

func (j *JIRA) ListAssignedIssues(showDescription bool) error {
	issues, err := j.getAssignedIssues()
	if err != nil {
		return err
	}
	for _, i := range issues {
		fmt.Printf("%v	%v\n", i.Key, i.Fields.Summary)
		if showDescription {
			fmt.Println(i.Fields.Description)
		}
	}
	return nil
}

func (j *JIRA) PickAssignedIssue() (*jira.Issue, error) {
	issues, err := j.getAssignedIssues()
	if err != nil {
		return nil, err
	}
	return j.pickIssue(issues)
}

func (j *JIRA) CreateBranchFromAssignedIssue() error {
	issue, err := j.PickAssignedIssue()
	if err != nil {
		return err
	}
	return j.CreateBranchFromIssue(issue, ".", false)
}

func (j *JIRA) CreateBranchFromIssue(issue *jira.Issue, repoDir string, forceNewBranch bool) error {
	git := core.MustInitGit(repoDir)
	git.Fetch()
	prefix := "chore"
	issueType := issue.Fields.Type.Name
	if issueType == "Bug" {
		prefix = "fix"
	} else if issueType == "Story" {
		prefix = "feat"
	}
	prefix = prefix + "/" + issue.Key + "/"
	err := git.CreateBranch(prefix + issue.Fields.Summary)
	if err != nil {
		if forceNewBranch || utils.AskForConfirmation("Failed to create branch. Force/overwrite?") {
			return git.ForceCreateBranch(prefix + " " + issue.Fields.Summary)
		}
		return nil
	}
	return nil
}

func (j *JIRA) sanitizeTransitionName(tr string) string {
	r := strings.NewReplacer("'", "", " ", "", "in", "")
	return r.Replace(strings.ToLower(tr))
}

func (j *JIRA) matchTransition(key, transitionName string) (jira.Transition, error) {
	trs, _, err := j.client.Issue.GetTransitions(key)
	transitionName = j.sanitizeTransitionName(transitionName)
	if err != nil {
		return jira.Transition{}, err
	}
	for _, tr := range j.cfg.JIRA.Transitions {
		if j.sanitizeTransitionName(tr.Alias) == transitionName {
			transitionName = tr.Name
		}
	}
	for _, tr := range trs {
		if j.sanitizeTransitionName(tr.Name) == transitionName {
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
	log.Printf("%v to be transitioned.", key)
	transition, err := j.matchTransition(key, transitionName)
	if err != nil {
		return err
	}
	res, err := j.client.Issue.DoTransition(key, transition.ID)
	if err != nil {
		j.logBody(res)
		return err
	}

	log.Printf("%v transitoned to %v", key, transition.Name)
	return nil
}

func (j *JIRA) getIssueKeyFromBranchOrAssigned() (string, error) {
	var key string
	if utils.InRepository() {
		key = core.InitGit().GetIssueKeyFromBranch()
	}
	if key == "" {
		log.Print("No issue key found in branch name. Fetching assigned issue(s).")
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

func (j *JIRA) MoveIssueToCurrentSprint(i *jira.Issue) error {
	sp, err := j.getActiveSprint()
	if err != nil {
		return err
	}
	_, err = j.client.Sprint.MoveIssuesToSprint(sp.ID, []string{i.Key})
	if err != nil {
		return err
	}
	log.Printf("%v moved to the active sprint.", i.Key)
	return nil
}

func (j *JIRA) CreateIssue(project, summary, description, transition string, reactive bool) error {
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
		fields.Labels = []string{"reactive"}
		fields.Unknowns = tcontainer.MarshalMap{"customfield_10100": 1}
	}

	i, res, err := j.client.Issue.Create(&jira.Issue{Fields: &fields})
	if err != nil {
		j.logBody(res)
		return err
	}
	log.Printf("%v created. %v", i.Key, strings.TrimRight(j.cfg.JIRA.Server, "/")+"/browse/"+i.Key)
	if transition != "" {
		if err = j.TransitionIssue(i.Key, transition); err != nil {
			return err
		}
	}
	if reactive {
		err = j.MoveIssueToCurrentSprint(i)
		if err != nil {
			return err
		}
		if utils.InRepository() && utils.AskForConfirmation("Checkout branch?") {
			i, _, err = j.client.Issue.Get(i.Key, &jira.GetQueryOptions{})
			if err != nil {
				return err
			}
			return j.CreateBranchFromIssue(i, ".", false)
		}
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

func (j *JIRA) openIssue(issue *jira.Issue, useBee bool) error {
	return j.OpenIssueFromKey(issue.Key, useBee)
}

func (j *JIRA) OpenIssueFromKey(key string, useBee bool) error {
	beeInstalled, err := utils.PathExists("/Applications/Bee.app")
	if err != nil {
		return nil
	}
	if useBee && beeInstalled {
		utils.OpenURI("bee://item?id=" + key)
		return nil
	}
	utils.OpenURI(j.cfg.JIRA.Server, "browse", key)
	return nil
}

func (j *JIRA) OpenIssue(key string, browser bool) error {
	key, err := j.getIssueIfNotSet(key)
	if err != nil {
		return err
	}
	return j.OpenIssueFromKey(key, browser)
}

func (j *JIRA) getIssueIfNotSet(key string) (string, error) {
	if key == "" {
		var err error
		key, err = j.getIssueKeyFromBranchOrAssigned()
		if err != nil {
			return "", err
		}
	}
	return key, nil
}

func (j *JIRA) CommentOnIssue(key, body string) error {
	key, err := j.getIssueIfNotSet(key)
	if err != nil {
		return err
	}
	_, res, err := j.client.Issue.AddComment(key, &jira.Comment{Body: body})
	if err != nil {
		j.logBody(res)
		return err
	}
	return err
}

func (j *JIRA) ViewIssue(key string) error {
	if key == "" {
		var err error
		key, err = j.getIssueKeyFromBranchOrAssigned()
		if err != nil {
			return nil
		}
	}
	i, res, err := j.client.Issue.Get(key, &jira.GetQueryOptions{})
	if err != nil {
		j.logBody(res)
		return err
	}
	details := `{{ "Key:" | faint }}		{{ .Key }}
{{ "Summary:" | faint }}	{{ .Fields.Summary }}
{{ "Assignee:" | faint }}	{{ if .Fields.Assignee }}{{ .Fields.Assignee.DisplayName }}{{ end }}
{{ "Reporter:" | faint }}	{{ .Fields.Reporter.DisplayName }}
{{ "Status:" | faint }}		{{ .Fields.Status.Name }}
{{ "Created:" | faint }}	{{ .Fields.Created }}
{{ "Updated:" | faint }}	{{ .Fields.Updated }}

{{ "Description:" | faint }}
{{ "-----------------------" | faint }}
{{ .Fields.Description | wordWrap }}

{{ "Comments:" | faint }}
{{ "-----------------------" | faint }}
{{ range $comment := .Fields.Comments.Comments }}{{ $comment.Body | wordWrap }}
{{ "↪" | faint }} {{ $comment.Author.DisplayName | faint }} {{ "(" | faint }}{{ $comment.Created | faint }}{{ ")" | faint }}

{{ end }}
`
	funcMaps := promptui.FuncMap
	funcMaps["wordWrap"] = utils.WordWrap
	t, err := template.New("records").Funcs(funcMaps).Parse(details)
	if err != nil {
		return err
	}
	return t.Execute(os.Stdout, i)
}

func (j *JIRA) pickIssue(issues []jira.Issue) (*jira.Issue, error) {
	if len(issues) == 0 {
		return nil, errors.New("no issue to pick")
	}
	if len(issues) == 1 {
		issue := issues[0]
		log.Printf("%v %v only available", issue.Key, issue.Fields.Summary)
		return &issue, nil
	}
	funcMaps := promptui.FuncMap
	funcMaps["wordWrap"] = utils.WordWrap

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "▶ {{ .Key }}	{{ .Fields.Summary }}",
		Inactive: "  {{ .Key }}	{{ .Fields.Summary }}",
		Selected: "▶ {{ .Key }}	{{ .Fields.Summary }}",
		Details: `
--------- Issue ----------
{{ "Key:" | faint }}	{{ .Key }}
{{ "Summary:" | faint }}	{{ .Fields.Summary }}
{{ "Type:" | faint }}	{{ .Fields.Type.Name }}
{{ "Assignee:" | faint }}	{{ if .Fields.Assignee }}{{ .Fields.Assignee.DisplayName }}{{ end }}
{{ "Reporter:" | faint }}	{{ .Fields.Reporter.DisplayName }}
{{ "Status:" | faint }}	{{ .Fields.Status.Name }}
{{ "Description:" | faint }}
{{ .Fields.Description | wordWrap }}
`,
		FuncMap: funcMaps,
	}

	searcher := func(input string, index int) bool {
		i := issues[index]
		name := strings.Replace(strings.ToLower(i.Key+i.Fields.Summary), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Size:              20,
		Label:             "Pick an issue",
		Items:             issues,
		Templates:         templates,
		Searcher:          searcher,
		StartInSearchMode: true,
	}
	i, _, err := prompt.Run()
	return &issues[i], err
}

func (j *JIRA) pickTransition(transitions []jira.Transition) (jira.Transition, error) {
	if len(transitions) == 0 {
		return jira.Transition{}, errors.New("no transition to pick")
	}
	if len(transitions) == 1 {
		return transitions[0], nil
	}
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}:",
		Active:   "▶ {{ .Name }}	{{ .ID }}",
		Inactive: "  {{ .Name }}	{{ .ID }}",
		Selected: "▶ {{ .Name }}	{{ .ID }}",
		Details:  "",
	}

	searcher := func(input string, index int) bool {
		i := transitions[index]
		name := strings.Replace(strings.ToLower(i.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Size:              20,
		Label:             "Pick an transition",
		Items:             transitions,
		Templates:         templates,
		Searcher:          searcher,
		StartInSearchMode: true,
	}
	i, _, err := prompt.Run()
	return transitions[i], err
}

package main

import (
	"fmt"
	"github.com/manifoldco/promptui"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"text/tabwriter"
)

type Git struct {
	cfg        *Configuration
}

func MustInitGit() *Git {
	return &Git{}
}

func (g *Git) GetCurrentRepositoryName() string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	result, err := cmd.Output()

	if err != nil {
		log.Fatalf("Failed to get repository: %v", err)
	}

	repositoryUri := string(result)
	return strings.TrimSuffix(path.Base(repositoryUri), path.Ext(repositoryUri))
}

func (g *Git) GetCurrentBranch() string {
	result, err := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD").Output()
	if err != nil {
		// if on jenkins the HEAD is usually detached, but you can infer the branch name.
		log.Printf("Could not get branch name from git: %v", err)
		log.Print("Trying to infer from environment variables.")
		return os.Getenv("BRANCH_NAME")
	}

	return strings.Trim(string(result), "\n ")
}

func (g *Git) GetTitleFromBranchName() string {
	branch := g.GetCurrentBranch()
	return strings.Replace(strings.Replace(strings.Replace(branch, "-", "_", 1), "-", " ", -1), "_", "-", -1)
}

func InRepository() bool {
	result, err := PathExists(".git")
	if err != nil {
		return false
	}
	return result
}

func (g *Git) CloneRepository(repository string) {
	log.Printf("Cloning: %v", repository)
	MustRunCmd("git", "clone", "git@github.com:benchlabs/"+repository+".git")
}

func (g *Git) Push(cfg *Configuration) {
	args := []string{"push", "--no-verify", "--set-upstream", "origin", g.GetCurrentBranch()}
	if cfg.Git.NoVerify {
		args = append(args, "--no-verify")
	}
	MustRunCmd("git", args...)
}

func (g *Git) UpdateRepository(repository string) {
	log.Printf("Updating: %v", repository)
	dir, _ := os.Getwd()
	os.Chdir(path.Join(dir, repository))
	MustRunCmd("git", "stash")
	MustRunCmd("git", "checkout", "master")
	MustRunCmd("git", "pull")
	os.Chdir(dir)
}

func (g *Git) SyncRepositories() {
	for _, m := range GetManifestRepository().GetAllActiveManifests() {
		g.syncRepository(m)
	}
}

func (g *Git) syncRepository(m Manifest) {
	repository := m.Repository
	repositoryExists, _ := PathExists(repository)
	if repositoryExists {
		g.UpdateRepository(repository)
	} else {
		g.CloneRepository(repository)
	}
}

func MustRunCmd(cmd string, args ...string) {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

func MustRunCmdWithOutput(cmd string, args ...string) string {
	command := exec.Command(cmd, args...)
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
	return string(output)
}

func RunCmdWithOutput(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	output, err := command.Output()
	return strings.Trim(string(output), "\n"), err
}

func (g *Git) Log() (commits []*GitCommit) {
	output := strings.Split(MustRunCmdWithOutput("git", "log", "--pretty=format:%h||~||%an||~||%s||~||%b|~~~~~|"), "|~~~~~|\n")
	for _, line := range output {
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, "||~||")
		commits = append(commits, &GitCommit{Hash:fields[0], Committer:fields[1], Subject:fields[2], Body:fields[3]})
	}
	return commits
}

func (g *Git) PendingChanges(cfg *Configuration, manifest *Manifest, previousVersion, currentVersion string, formatForSlack bool, noAt bool) {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output := MustRunCmdWithOutput("git", "log", "--first-parent", "--pretty=format:%h\t\t%an\t%s", previousVersion+"..."+currentVersion)
	if formatForSlack {
		re := g.GetIssueRegex()
		output = re.ReplaceAllString(output, "<https://"+cfg.JIRA.Server+"/browse/$1|$1>")
		re = g.GetPRRegex()
		output = re.ReplaceAllString(output, "<https://github.com/"+cfg.GitHub.Organization+"/"+manifest.Repository+"/pull/$2|PR#$2> ")
		re = regexp.MustCompile("(?m:^)([a-z0-9]{6,})")
		output = re.ReplaceAllString(output, "<https://github.com/"+cfg.GitHub.Organization+"/"+manifest.Repository+"/commit/$1|$1>")
	}
	fmt.Fprintln(table, output)
	table.Flush()
	if !noAt {
		committerSlackArr := g.committerSlackReference(cfg, previousVersion, currentVersion)
		if formatForSlack {
			fmt.Print("\n" + strings.Join(committerSlackArr, ", "))
		}
	}
}
func (g *Git) GetPRRegex() *regexp.Regexp {
	return regexp.MustCompile("(Merge pull request #)(\\d+) from \\w+/")
}
func (g *Git) GetIssueRegex() *regexp.Regexp {
	return regexp.MustCompile("([A-Z]{2,}-\\d+)")
}

type GitCommit struct {
	Hash, Committer, Subject, Body string
}

func (g *Git) PickCommit(commits []*GitCommit) (*GitCommit, error) {
	templates := &promptui.SelectTemplates{
		Label: "{{ . }}:",
		Active: "▶ {{ .Hash }}	{{ .Subject }}",
		Inactive: "  {{ .Hash }}	{{ .Subject }}",
		Selected: "▶ {{ .Hash }}	{{ .Subject }}",
		Details: `
{{ .Hash }}
{{ .Committer }}
{{ .Subject }}
{{ .Body }}
`,
	}

	searcher := func(input string, index int) bool {
		i := commits[index]
		name := strings.Replace(strings.ToLower(i.Subject), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Size:      20,
		Label:     "Pick commit",
		Items:     commits,
		Templates: templates,
		Searcher:  searcher,
	}
	i, _, err := prompt.Run()
	return commits[i], err
}

func (g *Git) FetchTags() {
	MustRunCmd("git", "fetch", "--tags")
}

func (g *Git) Fetch() {
	MustRunCmd("git", "fetch")
}

func (g *Git) sanitizeBranchName(name string) string {
	r2 := regexp.MustCompile("-+")
	return strings.Trim(r2.ReplaceAllString(g.GetIssueRegex().ReplaceAllString(name, "-"), "-"), "-")
}

func (g *Git) LogNotInMasterSubjects() []string {
	return strings.Split(MustRunCmdWithOutput("git", "log", "HEAD", "--not", "origin/master", "--no-merges", "--pretty=format:%s"), "\n")
}

func (g *Git) LogNotInMasterBody() string {
	return MustRunCmdWithOutput("git", "log", "HEAD", "--not", "origin/master", "--no-merges", "--pretty=format:-> %B")
}

func (g *Git) GetIssueKeyFromBranch() string {
	name, err := RunCmdWithOutput("git", "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return g.extractIssueKeyFromName(name)
}

func (g *Git) CommitWithIssueKey(cfg *Configuration, message string, extraArgs []string) {
	issueKey := g.GetIssueKeyFromBranch()
	args := []string{
		"commit", "-m", issueKey + " " + strings.Trim(message, " "),
	}
	if cfg.Git.NoVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, extraArgs...)
	MustRunCmd("git", args...)
}
func (g *Git) extractIssueKeyFromName(name string) string {
	return g.GetIssueRegex().FindString(name)
}

func (g *Git) CreateBranch(name string) {
	name = g.sanitizeBranchName(name)
	MustRunCmd("git", "checkout", "-b", name)
}

func (g *Git) CheckoutBranch() error {
	item, err := PickItem("Pick a branch", g.getBranches())
	if err != nil {
		return err
	}
	MustRunCmd("git", "checkout", item)
	return nil
}

func ForEachRepo(fn func() error) error {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	for _, value := range files {
		if !value.IsDir() {
			continue
		}

		err = os.Chdir(path.Join(wd, value.Name()))
		if err != nil {
			return err
		}
		if !InRepository() {
			continue
		}
		log.Printf(value.Name())
		err = fn()
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Git) getBranches() []string {
	output := MustRunCmdWithOutput("git", "branch", "--all", "--sort=-committerdate")
	var branches []string
	for _, b := range strings.Split(output, "\n") {
		b = strings.TrimPrefix(strings.Trim(b, " "), "* ")
		if b == "" {
			continue
		}
		branches = append(branches, b)
	}
	return branches
}

func (g *Git) committerSlackReference(cfg *Configuration, previousVersion string, currentVersion string) []string {
	committerMapping := make(map[string]string)
	for _, i := range cfg.Users {
		committerMapping[i.Name] = i.Slack
	}

	committersStdout := MustRunCmdWithOutput("git", "log", "--first-parent", "--pretty=format:%an", previousVersion+"..."+currentVersion)
	committersSlackMapping := make(map[string]string)
	for _, commiterName := range strings.Split(committersStdout, "\n") {
		slackUserName := committerMapping[commiterName]
		if slackUserName == "" {
			slackUserName = commiterName
		} else {
			slackUserName = "@" + slackUserName
		}
		committersSlackMapping[commiterName] = slackUserName
	}

	var committerSlackArr []string
	for _, v := range committersSlackMapping {
		committerSlackArr = append(committerSlackArr, v)
	}
	return committerSlackArr
}

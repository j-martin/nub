package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"text/tabwriter"
)

type git struct {
	cfg *Configuration
}

func Git() *git {
	return &git{}
}

func (g *git) GetCurrentRepositoryName() string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	result, err := cmd.Output()

	if err != nil {
		log.Fatalf("Failed to get repository: %v", err)
	}

	repositoryUri := string(result)
	return strings.TrimSuffix(path.Base(repositoryUri), path.Ext(repositoryUri))
}

func (g *git) GetCurrentBranch() string {
	result, err := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD").Output()
	if err != nil {
		// if on jenkins the HEAD is usually detached, but you can infer the branch name.
		log.Printf("Could not get branch name from git: %v", err)
		log.Print("Trying to infer from environment variables.")
		return os.Getenv("BRANCH_NAME")
	}

	return strings.Trim(string(result), "\n ")
}

func (g *git) inRepository() bool {
	result, err := pathExists(".git")
	if err != nil {
		return false
	}
	return result
}

func (g *git) CloneRepository(repository string) {
	log.Printf("Cloning: %v", repository)
	MustRunCmd("git", "clone", "git@github.com:benchlabs/"+repository+".git")
}

func (g *git) Push(cfg *Configuration) {
	args := []string{"push", "--no-verify", "--set-upstream", "origin", g.GetCurrentBranch()}
	if cfg.Git.NoVerify {
		args = append(args, "--no-verify")
	}
	MustRunCmd("git", args...)
}

func (g *git) UpdateRepository(repository string) {
	log.Printf("Updating: %v", repository)
	dir, _ := os.Getwd()
	os.Chdir(path.Join(dir, repository))
	MustRunCmd("git", "stash")
	MustRunCmd("git", "checkout", "master")
	MustRunCmd("git", "pull")
	os.Chdir(dir)
}

func (g *git) SyncRepositories() {
	for _, m := range GetAllActiveManifests() {
		g.syncRepository(m)
	}
}

func (g *git) syncRepository(m Manifest) {
	repository := m.Repository
	repositoryExists, _ := pathExists(repository)
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
func (g *git) PendingChanges(cfg *Configuration, manifest Manifest, previousVersion, currentVersion string, formatForSlack bool, noAt bool) {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output := MustRunCmdWithOutput("git", "log", "--first-parent", "--pretty=format:%h\t\t%an\t%s", previousVersion+"..."+currentVersion)
	if formatForSlack {
		re := regexp.MustCompile("([A-Z]{2,}-\\d+)")
		output = re.ReplaceAllString(output, "<https://"+cfg.JIRA.Server+"/browse/$1|$1>")
		re = regexp.MustCompile("(Merge pull request #)(\\d+) from \\w+/")
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

func (g *git) FetchTags() {
	MustRunCmd("git", "fetch", "--tags")
}

func (g *git) Fetch() {
	MustRunCmd("git", "fetch")
}

func (g *git) sanitizeBranchName(name string) string {
	r := regexp.MustCompile("[^a-zA-Z0-9]+")
	r2 := regexp.MustCompile("-+")
	return strings.Trim(r2.ReplaceAllString(r.ReplaceAllString(name, "-"), "-"), "-")
}

func (g *git) LogNotInMasterSubjects() []string {
	return strings.Split(MustRunCmdWithOutput("git", "log", "HEAD", "--not", "origin/master", "--no-merges", "--pretty=format:%s"), "\n")
}

func (g *git) LogNotInMasterBody() string {
	return MustRunCmdWithOutput("git", "log", "HEAD", "--not", "origin/master", "--no-merges", "--pretty=format:-> %B")
}

func (g *git) GetIssueKeyFromBranch() string {
	name, err := RunCmdWithOutput("git", "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return g.extractIssueKeyFromName(name)
}

func (g *git) CommitWithIssueKey(cfg *Configuration, message string, extraArgs []string) {
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
func (g *git) extractIssueKeyFromName(name string) string {
	r := regexp.MustCompile("^[A-Z]+-\\d+")
	return r.FindString(name)
}

func (g *git) CreateBranch(name string) {
	name = g.sanitizeBranchName(name)
	MustRunCmd("git", "checkout", "-b", name)
}

func (g *git) CheckoutBranch() error {
	item, err := pickItem("Pick a branch", g.getBranches())
	if err != nil {
		return err
	}
	MustRunCmd("git", "checkout", item)
	return nil
}

func (g *git) getBranches() []string {
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

func (g *git) committerSlackReference(cfg *Configuration, previousVersion string, currentVersion string) []string {
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

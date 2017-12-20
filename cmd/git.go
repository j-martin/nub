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

func GetCurrentRepositoryName() string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	result, err := cmd.Output()

	if err != nil {
		log.Fatalf("Failed to get repository: %v", err)
	}

	repositoryUri := string(result)
	return strings.TrimSuffix(path.Base(repositoryUri), path.Ext(repositoryUri))
}

func GetCurrentBranch() string {
	result, err := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD").Output()
	if err != nil {
		// if on jenkins the HEAD is usually detached, but you can infer the branch name.
		log.Printf("Could not get branch name from git: %v", err)
		log.Print("Trying to infer from environment variables.")
		return os.Getenv("BRANCH_NAME")
	}

	return strings.Trim(string(result), "\n ")
}

func inRepository() bool {
	result, err := pathExists(".git")
	if err != nil {
		return false
	}
	return result
}

func CloneRepository(repository string) {
	log.Printf("Cloning: %v", repository)
	MustRunCmd("git", "clone", "git@github.com:benchlabs/"+repository+".git")
}

func GitPush(cfg *Configuration) {
	args := []string{"push", "--no-verify", "--set-upstream", "origin", GetCurrentBranch()}
	if cfg.Git.NoVerify {
		args = append(args, "--no-verify")
	}
	MustRunCmd("git", args...)
}

func UpdateRepository(repository string) {
	log.Printf("Updating: %v", repository)
	dir, _ := os.Getwd()
	os.Chdir(path.Join(dir, repository))
	MustRunCmd("git", "stash")
	MustRunCmd("git", "checkout", "master")
	MustRunCmd("git", "pull")
	os.Chdir(dir)
}

func SyncRepositories() {
	for _, m := range GetAllActiveManifests() {
		syncRepository(m)
	}
}

func syncRepository(m Manifest) {
	repository := m.Repository
	repositoryExists, _ := pathExists(repository)
	if repositoryExists {
		UpdateRepository(repository)
	} else {
		CloneRepository(repository)
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
func PendingChanges(cfg *Configuration, manifest Manifest, previousVersion, currentVersion string, formatForSlack bool, noAt bool) {
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
		committerSlackArr := committerSlackReference(cfg, previousVersion, currentVersion)
		if formatForSlack {
			fmt.Print("\n" + strings.Join(committerSlackArr, ", "))
		}
	}
}

func FetchTags() {
	MustRunCmd("git", "fetch", "--tags")
}

func sanitizeBranchName(name string) string {
	r := regexp.MustCompile("[^a-zA-Z0-9]+")
	r2 := regexp.MustCompile("-+")
	return strings.Trim(r2.ReplaceAllString(r.ReplaceAllString(name, "-"), "-"), "-")
}

func LogNotInMasterSubjects() []string {
	return strings.Split(MustRunCmdWithOutput("git", "log", "HEAD", "--not", "master", "--no-merges", "--pretty=format:%s"), "\n")
}

func LogNotInMasterBody() string {
	return MustRunCmdWithOutput("git", "log", "HEAD", "--not", "master", "--no-merges", "--pretty=format:-> %B")
}

func GetIssueKeyFromBranch() string {
	name, err := RunCmdWithOutput("git", "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return extractIssueKeyFromName(name)
}

func CommitWithIssueKey(cfg *Configuration, message string, extraArgs []string) {
	issueKey := GetIssueKeyFromBranch()
	args := []string{
		"commit", "-m", issueKey + " " + strings.Trim(message, " "),
	}
	if cfg.Git.NoVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, extraArgs...)
	MustRunCmd("git", args...)
}
func extractIssueKeyFromName(name string) string {
	r := regexp.MustCompile("^[A-Z]+-\\d+")
	return r.FindString(name)
}

func CreateBranch(name string) {
	name = sanitizeBranchName(name)
	MustRunCmd("git", "checkout", "-b", name)
}

func CheckoutBranch() error {
	item, err := pickItem("Pick a branch", getBranches())
	if err != nil {
		return err
	}
	MustRunCmd("git", "checkout", item)
	return nil
}

func getBranches() []string {
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

func committerSlackReference(cfg *Configuration, previousVersion string, currentVersion string) []string {
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

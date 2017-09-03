package main

import (
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
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
	runCmd("git", "clone", "git@github.com:benchlabs/"+repository+".git")
}

func UpdateRepository(repository string) {
	log.Printf("Updating: %v", repository)
	dir, _ := os.Getwd()
	os.Chdir(path.Join(dir, repository))
	runCmd("git", "stash")
	runCmd("git", "checkout", "master")
	runCmd("git", "pull")
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

func runCmd(cmd string, args ...string) {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

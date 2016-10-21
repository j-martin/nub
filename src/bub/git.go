package main

import (
	"os/exec"
	"log"
	"strings"
	"path"
)

//TODO: Clone/Update All Active Projects

func GetCurrentRepositoryName() (string) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	result, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	repositoryUri := string(result)
	return strings.TrimSuffix(path.Base(repositoryUri), path.Ext(repositoryUri))

}


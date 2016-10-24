package main

import (
	"log"
	"net/url"
	"os/exec"
	"strings"
)

func OpenURI(uriSegments ...string) {
	uri := strings.Join(uriSegments, "/")
	log.Printf("Opening: %v", uri)
	exec.Command("open", uri).Run()
}

func OpenGH(m Manifest, p string) {
	base := "https://github.com/BenchLabs"
	OpenURI(base, m.Repository, p)
}

func OpenJenkins(m Manifest, p string) {
	base := "https://jenkins.example.com/job/BenchLabs/job"
	OpenURI(base, m.Repository, p)
}

func OpenSplunk(m Manifest, isStaging bool) {
	base := "https://splunk.example.com/en-US/app/search/search/?q=search%20sourcetype%3D"
	var sourceType []string
	if isStaging {
		sourceType = append(sourceType, "staging")
	}
	sourceType = append(sourceType, m.Name, "hec")
	OpenURI(base + strings.Join(sourceType, "-"))
}

func OpenCircle(m Manifest, getBranch bool) {
	base := "https://circleci.com/gh/BenchLabs"
	if getBranch {
		currentBranch := url.QueryEscape(GetCurrentBranch())
		OpenURI(base, m.Repository, "tree", currentBranch)
	} else {
		OpenURI(base, m.Repository)
	}
}

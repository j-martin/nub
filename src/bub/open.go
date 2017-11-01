package main

import (
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

func openURI(uriSegments ...string) {
	uri := strings.Join(uriSegments, "/")
	log.Printf("Opening: %v", uri)
	if runtime.GOOS == "darwin" {
		exec.Command("open", uri).Run()
	} else if runtime.GOOS == "linux" {
		exec.Command("xdg-open", uri).Run()
	} else {
		log.Fatal("Could not open the link automatically.")
	}
}

func openGH(m Manifest, p string) {
	base := "https://github.com/BenchLabs"
	openURI(base, m.Repository, p)
}

func openJenkins(m Manifest, p string) {
	base := "https://jenkins.example.com/job/BenchLabs/job"
	openURI(base, m.Repository, "job", m.Branch, p)
}

func openSplunk(m Manifest, isStaging bool) {
	base := "https://splunk.example.com/en-US/app/search/search/?dispatch.sample_ratio=1&earliest=rt-1h&latest=rtnow&q=search%20sourcetype%3D"
	var sourceType string
	if isStaging {
		sourceType = "staging"
	} else {
		sourceType = "pro"
	}
	sourceType = sourceType + "-" + m.Name + "*"
	openURI(base + sourceType)
}

func openCircle(m Manifest, getBranch bool) {
	base := "https://circleci.com/gh/BenchLabs"
	if getBranch {
		currentBranch := url.QueryEscape(GetCurrentBranch())
		openURI(base, m.Repository, "tree", currentBranch)
	} else {
		openURI(base, m.Repository)
	}
}

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

func openGH(cfg Configuration, m Manifest, p string) {
	openURI("https://github.com/", cfg.GitHub.Organization, m.Repository, p)
}

func openJenkins(cfg Configuration, m Manifest, p string) {
	openURI(cfg.Jenkins.Server, "/job/BenchLabs/job", m.Repository, "job", m.Branch, p)
}

func openSplunk(cfg Configuration, m Manifest, isStaging bool) {
	base := cfg.Splunk.Server +
		"/en-US/app/search/search/?dispatch.sample_ratio=1&earliest=rt-1h&latest=rtnow&q=search%20sourcetype%3D"
	var sourceType string
	if isStaging {
		sourceType = "staging"
	} else {
		sourceType = "pro"
	}
	sourceType = sourceType + "-" + m.Name + "*"
	openURI(base + sourceType)
}

func openCircle(cfg Configuration, m Manifest, getBranch bool) {
	base := "https://circleci.com/gh/" + cfg.GitHub.Organization
	if getBranch {
		currentBranch := url.QueryEscape(GetCurrentBranch())
		openURI(base, m.Repository, "tree", currentBranch)
	} else {
		openURI(base, m.Repository)
	}
}

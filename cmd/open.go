package main

import (
	"github.com/pkg/errors"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

func openURI(uriSegments ...string) error {
	uri := strings.Join(uriSegments, "/")
	log.Printf("Opening: %v", uri)
	if runtime.GOOS == "darwin" {
		return exec.Command("open", uri).Run()
	} else if runtime.GOOS == "linux" {
		return exec.Command("xdg-open", uri).Run()
	}
	return errors.New("could not open the link automatically")
}

func openGH(cfg *Configuration, m Manifest, p string) error {
	return openURI("https://github.com/", cfg.GitHub.Organization, m.Repository, p)
}

func openJenkins(cfg *Configuration, m Manifest, p string) error {
	return openURI(cfg.Jenkins.Server, "/job/BenchLabs/job", m.Repository, "job", m.Branch, p)
}

func openSplunk(cfg *Configuration, m Manifest, isStaging bool) error {
	base := cfg.Splunk.Server +
		"/en-US/app/search/search/?dispatch.sample_ratio=1&earliest=rt-1h&latest=rtnow&q=search%20sourcetype%3D"
	var sourceType string
	if isStaging {
		sourceType = "staging"
	} else {
		sourceType = "pro"
	}
	sourceType = sourceType + "-" + m.Name + "*"
	return openURI(base + sourceType)
}

func openCircle(cfg *Configuration, m Manifest, getBranch bool) error {
	base := "https://circleci.com/gh/" + cfg.GitHub.Organization
	if getBranch {
		currentBranch := url.QueryEscape(Git().GetCurrentBranch())
		return openURI(base, m.Repository, "tree", currentBranch)
	}
	return openURI(base, m.Repository)
}

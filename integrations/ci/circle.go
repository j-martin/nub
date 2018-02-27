package ci

import (
	"errors"
	"fmt"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/utils"
	"github.com/jszwedko/go-circleci"
	"log"
	"net/url"
	"os"
	"time"
)

type Circle struct {
	cfg    *core.Configuration
	client *circleci.Client
}

func MustInitCircle(cfg *core.Configuration) *Circle {
	token := os.Getenv("CIRCLE_TOKEN")
	if token == "" && cfg.Circle.Token == "" {
		log.Fatal("Please set the CircleCi token in your keychain or set with the CIRCLE_TOKEN environment variable.")
	} else if cfg.Circle.Token != "" {
		token = cfg.Circle.Token
	}
	return &Circle{cfg, &circleci.Client{Token: token}}
}

func OpenCircle(cfg *core.Configuration, m *core.Manifest, getBranch bool) error {
	base := "https://circleci.com/gh/" + cfg.GitHub.Organization
	if getBranch {
		currentBranch := url.QueryEscape(core.InitGit().GetCurrentBranch())
		return utils.OpenURI(base, m.Repository, "tree", currentBranch)
	}
	return utils.OpenURI(base, m.Repository)
}

func (c *Circle) TriggerAndWaitForSuccess(m *core.Manifest) error {
	build, err := c.client.Build(c.cfg.GitHub.Organization, m.Repository, m.Branch)
	if err != nil {
		return err
	}

	log.Printf("Triggered build: %s", build.BuildURL)

	time.Sleep(1 * time.Second)

	for {
		build, err = c.client.GetBuild(c.cfg.GitHub.Organization, m.Repository, build.BuildNum)
		if err != nil {
			return err
		}

		if build.Lifecycle == "finished" || build.Status == "not_run" || build.Lifecycle == "not_running" {
			break
		}

		log.Printf("Current lifecycle state: %s, waiting 20s...", build.Lifecycle)
		time.Sleep(20 * time.Second)
	}

	if build.Outcome == "success" {
		log.Print("The build succeeded!")
		return nil
	} else {
		return errors.New(fmt.Sprintf("the build failed: %s, %s", build.Outcome, build.BuildURL))
	}
}

func (c *Circle) CheckBuildStatus(m *core.Manifest) error {
	head, err := core.MustInitGit(".").CurrentHEAD()
	if err != nil {
		return err
	}
	log.Printf("Commit: %v", head)
	for {
		b, err := c.checkBuildStatus(head, m)
		if err != nil {
			return err
		}
		if utils.Contains(b.Status, "success", "fixed", "no_tests") {
			log.Printf("Status: '%v', The build is done. %v", b.Status, b.BuildURL)
			return nil
		}
		if utils.Contains(b.Status, "failed", "canceled", "infrastructure_fail", "timedout") {
			log.Fatalf("Status: '%v'. Aborting. %v", b.Status, b.BuildURL)
		}
		log.Printf("Status: '%v', waiting 10s. %v", b.Status, b.BuildURL)
		time.Sleep(10 * time.Second)
	}
	return nil
}

func (c *Circle) checkBuildStatus(head string, m *core.Manifest) (*circleci.Build, error) {
	builds, err := c.client.ListRecentBuildsForProject(c.cfg.GitHub.Organization, m.Repository, m.Branch, "", 50, 0)
	if err != nil {
		return nil, err
	}
	for _, b := range builds {
		commit := b.AllCommitDetails[len(b.AllCommitDetails)-1].Commit
		if commit == head {
			return b, nil
		}
	}
	return nil, errors.New("no build found for the commit")
}

package main

import (
	"github.com/jszwedko/go-circleci"
	"log"
	"os"
	"time"
)

func triggerAndWaitForSuccess(cfg Configuration, m Manifest) {

	token := os.Getenv("CIRCLE_TOKEN")

	if token == "" && cfg.Circle.Token == "" {
		log.Fatal("Please set the CircleCi token in your ~/.config/bub/config.yml or set with the CIRCLE_TOKEN environment variable.")
	} else if cfg.Circle.Token != "" {
		token = cfg.Circle.Token
	}

	client := circleci.Client{Token: token}
	build, err := client.Build(cfg.Github.Organization, m.Repository, m.Branch)
	if err != nil {
		log.Fatal("The job could not be triggered.", err)
	}

	log.Printf("Triggered build: %s", build.BuildURL)

	time.Sleep(1 * time.Second)

	for {
		build, err = client.GetBuild(cfg.Github.Organization, m.Repository, build.BuildNum)
		if err != nil {
			log.Fatal("The job status could not be fetched.", err)
		}

		if build.Lifecycle == "finished" || build.Status == "not_run" || build.Lifecycle == "not_running" {
			break
		}

		log.Printf("Current lifecycle state: %s, waiting 20s...", build.Lifecycle)
		time.Sleep(20 * time.Second)
	}

	if build.Outcome == "success" {
		log.Print("The build succeeded!")
	} else {
		log.Fatalf("The build failed: %s, %s", build.Outcome, build.BuildURL)
	}

}

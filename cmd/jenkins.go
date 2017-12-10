package main

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

func getJobName(m Manifest) string {
	return strings.Join([]string{"BenchLabs", "job", m.Repository, "job", m.Branch}, "/")
}

func getClient(cfg Configuration) *gojenkins.Jenkins {
	checkServerConfig(cfg.Jenkins)
	loadCredentials("Jenkins", &cfg.Jenkins)
	client, err := gojenkins.CreateJenkins(cfg.Jenkins.Server, cfg.Jenkins.Username, cfg.Jenkins.Password).Init()
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func getJob(cfg Configuration, m Manifest) *gojenkins.Job {
	client := getClient(cfg)
	uri := getJobName(m)
	job, err := client.GetJob(uri)
	if err != nil {
		log.Fatalf("Failed to fetch job details. error: %s", err)
	}
	return job
}

func getLastBuild(cfg Configuration, m Manifest) *gojenkins.Build {
	log.Printf("Fetching last build for '%v' '%v'.", m.Repository, m.Branch)
	lastBuild, err := getJob(cfg, m).GetLastBuild()
	if err != nil {
		log.Fatalf("Failed to fetch build details. error: %s", err)
	}
	log.Printf(lastBuild.GetUrl())
	return lastBuild
}

func getArtifacts(cfg Configuration, m Manifest) {
	log.Print("Fetching artifacts.")
	artifacts := getLastBuild(cfg, m).GetArtifacts()
	dir, _ := ioutil.TempDir("", strings.Join([]string{m.Repository, m.Branch}, "-"))
	for _, artifact := range artifacts {
		if !strings.Contains(artifact.FileName, ".png") {
			artifactPath := path.Join(dir, artifact.FileName)
			log.Println(artifactPath)
			artifact.Save(artifactPath)
		} else {
			log.Println(cfg.Jenkins.Server + artifact.Path)
		}
	}
}

func showConsoleOutput(cfg Configuration, m Manifest) {
	var lastChar int
	for {
		build, err := getJob(cfg, m).GetLastBuild()
		if lastChar == 0 {
			log.Print(build.GetUrl())
		}
		if err != nil {
			log.Fatalf("Could not find the last build. make sure it was triggered at least once: %v", err)
		}
		consoleOutput := build.GetConsoleOutput()
		for i, char := range consoleOutput {
			if i > lastChar {
				fmt.Print(string(char))
			}
		}
		lastChar = len(consoleOutput) - 1
		if !build.IsRunning() {
			if !build.IsGood() {
				log.Fatal("The job failed on jenkins.")
			}
			break
		}
		time.Sleep(2 * time.Second)
	}
}

func buildJob(cfg Configuration, m Manifest, async bool, force bool) {
	jobName := getJobName(m)
	job := getJob(cfg, m)
	lastBuild, err := job.GetLastBuild()
	if err == nil && lastBuild.IsRunning() && !force {
		log.Fatal("A build for this job is already running pass '--force' to trigger the build.")
	} else if err != nil && err.Error() != "404" {
		log.Fatalf("Failed to get last build status: %v", err)
	}

	job.InvokeSimple(nil)
	log.Printf("Build triggered: %v/job/%v wating for the job to start.", cfg.Jenkins.Server, jobName)

	if async {
		return
	}

	for {
		newBuild, err := getJob(cfg, m).GetLastBuild()
		if err == nil && (lastBuild == nil || (lastBuild.GetUrl() != newBuild.GetUrl())) {
			os.Stderr.WriteString("\n")
			break
		} else if err != nil && err.Error() != "404" {
			log.Fatalf("Failed to get build status: %v", err)
		}
		os.Stderr.WriteString(".")
		time.Sleep(2 * time.Second)
	}
	showConsoleOutput(cfg, m)
}

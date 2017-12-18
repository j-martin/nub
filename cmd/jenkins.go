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

type Jenkins struct {
	cfg    *Configuration
	client *gojenkins.Jenkins
}

func (j *Jenkins) getJobName(m Manifest) string {
	return path.Join(j.cfg.GitHub.Organization, "job", m.Repository, "job", m.Branch)
}

func MustInitJenkins(cfg *Configuration) *Jenkins {
	checkServerConfig(cfg.Jenkins.Server)
	loadCredentials("Jenkins", &cfg.Jenkins.Username, &cfg.Jenkins.Password)
	jenkins := gojenkins.CreateJenkins(cfg.Jenkins.Server, cfg.Jenkins.Username, cfg.Jenkins.Password)
	client, err := jenkins.Init()
	if err != nil {
		log.Fatal(err)
	}
	return &Jenkins{cfg: cfg, client: client}
}

func (j *Jenkins) getJob(m Manifest) *gojenkins.Job {
	uri := j.getJobName(m)
	job, err := j.client.GetJob(uri)
	if err != nil {
		log.Fatalf("Failed to fetch job details. error: %s", err)
	}
	return job
}

func (j *Jenkins) getLastBuild(m Manifest) *gojenkins.Build {
	log.Printf("Fetching last build for '%v' '%v'.", m.Repository, m.Branch)
	lastBuild, err := j.getJob(m).GetLastBuild()
	if err != nil {
		log.Fatalf("Failed to fetch build details. error: %s", err)
	}
	log.Printf(lastBuild.GetUrl())
	return lastBuild
}

func (j *Jenkins) getArtifacts(m Manifest) error {
	log.Print("Fetching artifacts.")
	artifacts := j.getLastBuild(m).GetArtifacts()
	dir, err := ioutil.TempDir("", strings.Join([]string{m.Repository, m.Branch}, "-"))
	if err != nil {
		return nil
	}
	for _, artifact := range artifacts {
		if !strings.Contains(artifact.FileName, ".png") {
			artifactPath := path.Join(dir, artifact.FileName)
			log.Println(artifactPath)
			_, err := artifact.Save(artifactPath)
			if err != nil {
				return err
			}
		} else {
			log.Println(j.cfg.Jenkins.Server + artifact.Path)
		}
	}
	return nil
}

func (j *Jenkins) showConsoleOutput(m Manifest) {
	var lastChar int
	for {
		build, err := j.getJob(m).GetLastBuild()
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

func (j *Jenkins) buildJob(m Manifest, async bool, force bool) {
	jobName := j.getJobName(m)
	job := j.getJob(m)
	lastBuild, err := job.GetLastBuild()
	if err == nil && lastBuild.IsRunning() && !force {
		log.Fatal("A build for this job is already running pass '--force' to trigger the build.")
	} else if err != nil && err.Error() != "404" {
		log.Fatalf("Failed to get last build status: %v", err)
	}

	job.InvokeSimple(nil)
	log.Printf("Build triggered: %v/job/%v wating for the job to start.", j.cfg.Jenkins.Server, jobName)

	if async {
		return
	}

	for {
		newBuild, err := j.getJob(m).GetLastBuild()
		if err == nil && (lastBuild == nil || (lastBuild.GetUrl() != newBuild.GetUrl())) {
			os.Stderr.WriteString("\n")
			break
		} else if err != nil && err.Error() != "404" {
			log.Fatalf("Failed to get build status: %v", err)
		}
		os.Stderr.WriteString(".")
		time.Sleep(2 * time.Second)
	}
	j.showConsoleOutput(m)
}

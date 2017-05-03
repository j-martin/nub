package main

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"time"
	"os"
)

func GetJobName(m Manifest) string {
	return strings.Join([]string{"BenchLabs", "job", m.Repository, "job", m.Branch}, "/")
}

func GetClient(cfg Configuration) *gojenkins.Jenkins {
	if cfg.Jenkins.Server == "" {
		log.Fatal("server cannot be empty, make sure the config file is properly configured. run 'bub config'.")
	}
	if strings.HasPrefix(cfg.Jenkins.Username, "<") ||
		cfg.Jenkins.Username == "" || cfg.Jenkins.Password == "" {
		log.Fatal("please set your jenkins credentials. run 'bub config'.")
	}
	client, err := gojenkins.CreateJenkins(cfg.Jenkins.Server, cfg.Jenkins.Username, cfg.Jenkins.Password).Init()
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func GetJob(cfg Configuration, m Manifest) *gojenkins.Job {
	client := GetClient(cfg)
	uri := GetJobName(m)
	job, err := client.GetJob(uri)
	if err != nil {
		log.Fatalf("failed to fetch job details. error: %s", err)
	}
	return job
}

func GetLastBuild(cfg Configuration, m Manifest) *gojenkins.Build {
	log.Printf("fetching last build for '%v' '%v'.", m.Repository, m.Branch)
	lastBuild, err := GetJob(cfg, m).GetLastBuild()
	if err != nil {
		log.Fatalf("failed to fetch build details. error: %s", err)
	}
	log.Printf(lastBuild.GetUrl())
	return lastBuild
}

func GetArtifacts(cfg Configuration, m Manifest) {
	log.Print("fetching artifacts.")
	artifacts := GetLastBuild(cfg, m).GetArtifacts()
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

func ShowConsoleOutput(cfg Configuration, m Manifest) {
	var lastChar int
	for {
		build, err := GetJob(cfg, m).GetLastBuild()
		if err != nil {
			log.Fatal(err)
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
				log.Fatal("the job failed on jenkins.")
			}
			break
		}
		time.Sleep(2 * time.Second)
	}
}

func BuildJob(cfg Configuration, m Manifest) {
	jobName := GetJobName(m)
	job := GetJob(cfg, m)
	lastBuild, err := job.GetLastBuild()
	if err != nil {
		log.Fatalf("failed to get job status: %v", err)
	}

	job.InvokeSimple(nil)
	log.Printf("job triggered: %v, wating for the job to start.", jobName)
	for {
		newBuild, err := GetJob(cfg, m).GetLastBuild()
		if err != nil {
			log.Fatalf("failed to get job status: %v", err)
		}
		os.Stderr.WriteString(".")
		if lastBuild.GetUrl() != newBuild.GetUrl() {
			os.Stderr.WriteString("\n")
			break
		}
		time.Sleep(2 * time.Second)
	}
	ShowConsoleOutput(cfg, m)
}

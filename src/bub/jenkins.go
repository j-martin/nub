package main

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"io/ioutil"
	"log"
	"path"
	"strings"
)

func GetJobName(m Manifest) string {
	return strings.Join([]string{"BenchLabs", "job", m.Repository, "job", m.Branch}, "/")
}

func GetClient(cfg Configuration) *gojenkins.Jenkins {
	if cfg.Jenkins.Server == "" {
		log.Fatal("Server cannot be empty, make sure the config file is properly configured. Run 'bub config'.")
	}
	if strings.HasPrefix(cfg.Jenkins.Username, "<") ||
		cfg.Jenkins.Username == "" ||
		cfg.Jenkins.Password == "" {
		log.Fatal("Please set your jenkins credentials. Run 'bub config'.")
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
		log.Fatalf("Failed to fetch job details. Error: %s", err)
	}
	return job
}

func GetLastBuild(cfg Configuration, m Manifest) *gojenkins.Build {
	log.Printf("Fetching last build for '%v' '%v'.", m.Repository, m.Branch)
	lastBuild, err := GetJob(cfg, m).GetLastBuild()
	if err != nil {
		log.Fatalf("Failed to fetch build details. Error: %s", err)
	}
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
	fmt.Println(GetLastBuild(cfg, m).GetConsoleOutput())
}

func BuildJob(cfg Configuration, m Manifest) {
	jobName := GetJobName(m)
	GetJob(cfg, m).InvokeSimple(nil)
	log.Printf("Job Triggered: %v", jobName)
}

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
		log.Fatal("Server cannot be empty, make sure the config file is properly configured.")
	}
	client, _ := gojenkins.CreateJenkins(cfg.Jenkins.Server, cfg.Jenkins.Username, cfg.Jenkins.Password).Init()
	return client
}

func GetJob(cfg Configuration, m Manifest) *gojenkins.Job {
	client := GetClient(cfg)
	uri := GetJobName(m)
	job, _ := client.GetJob(uri)
	return job
}

func GetLastBuild(cfg Configuration, m Manifest) *gojenkins.Build {
	log.Printf("Fetching last build for '%v' '%v'.", m.Repository, m.Branch)
	lastBuild, _ := GetJob(cfg, m).GetLastBuild()
	return lastBuild
}

func GetArtifacts(cfg Configuration, m Manifest) {
	log.Print("Fetching artifacts.")
	artifacts := GetLastBuild(cfg, m).GetArtifacts()
	dir, _ := ioutil.TempDir("", strings.Join([]string{m.Repository, m.Branch}, "-"))
	for _, a := range artifacts {
		if !strings.Contains(a.FileName, ".png") {
			artifactPath := path.Join(dir, a.FileName)
			log.Println(artifactPath)
			a.Save(artifactPath)
		} else {
			log.Println(cfg.Jenkins.Server + a.Path)
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

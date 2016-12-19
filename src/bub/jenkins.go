package main

import (
	"github.com/bndr/gojenkins"
	"log"
	"strings"
	"io/ioutil"
	"path"
)

func GetLastBuild(cfg Configuration, m Manifest) *gojenkins.Build {
	if cfg.Jenkins.Server == "" {
		log.Fatal("Server cannot be empty, make sure the config file is properly configured.")
	}
	client, _ := gojenkins.CreateJenkins(cfg.Jenkins.Server, cfg.Jenkins.Username, cfg.Jenkins.Password).Init()
	log.Printf("Fetching last build for '%v' '%v'.", m.Repository, m.Branch)
	uri := strings.Join([]string{"BenchLabs", "job", m.Repository, "job", m.Branch}, "/")
	job, _ := client.GetJob(uri)
	lastBuild, _ := job.GetLastBuild()
	return lastBuild
}

func ListArtifacts(cfg Configuration, m Manifest) {
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

func ShowJobs(cfg Configuration, m Manifest) {
	log.Print(GetLastBuild(cfg, m).GetConsoleOutput())
}


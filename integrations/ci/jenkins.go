package ci

import (
	"fmt"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/utils"
	"github.com/bndr/gojenkins"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

type Jenkins struct {
	cfg      *core.Configuration
	manifest *core.Manifest
	client   *gojenkins.Jenkins
}

func (j *Jenkins) getJobName() string {
	return path.Join(j.cfg.GitHub.Organization, "job", j.manifest.Repository, "job", j.manifest.Branch)
}

func MustInitJenkins(cfg *core.Configuration, m *core.Manifest) *Jenkins {
	core.CheckServerConfig(cfg.Jenkins.Server)
	mustLoadJenkinsCredentials(cfg)
	jenkins := gojenkins.CreateJenkins(cfg.Jenkins.Server, cfg.Jenkins.Username, cfg.Jenkins.Password)
	client, err := jenkins.Init()
	if err != nil {
		log.Fatal(err)
	}
	return &Jenkins{cfg: cfg, client: client, manifest: m}
}

func mustLoadJenkinsCredentials(cfg *core.Configuration) {
	err := core.LoadCredentials("Jenkins", &cfg.Jenkins.Username, &cfg.Jenkins.Password, cfg.ResetCredentials)
	if err != nil {
		log.Fatalf("Failed to set JIRA credentials: %v", err)
	}
}

func MustSetupJenkins(cfg *core.Configuration) {
	utils.Prompt("Log into Jenkins, click on your username (top right corner), go to Configure, click on 'Show API Token...'.\n" +
		"Theses are you username and password. Continue?")
	utils.OpenURI(cfg.Jenkins.Server)
	mustLoadJenkinsCredentials(cfg)
}

func (j *Jenkins) getJob() *gojenkins.Job {
	uri := j.getJobName()
	job, err := j.client.GetJob(uri)
	if err != nil {
		log.Fatalf("Failed to fetch job details. error: %s", err)
	}
	return job
}

func (j *Jenkins) getLastBuild() *gojenkins.Build {
	log.Printf("Fetching last build for '%v' '%v'.", j.manifest.Repository, j.manifest.Branch)
	lastBuild, err := j.getJob().GetLastBuild()
	if err != nil {
		log.Fatalf("Failed to fetch build details. error: %s", err)
	}
	log.Printf(lastBuild.GetUrl())
	return lastBuild
}

func (j *Jenkins) GetArtifacts() error {
	log.Print("Fetching artifacts.")
	artifacts := j.getLastBuild().GetArtifacts()
	dir, err := ioutil.TempDir("", strings.Join([]string{j.manifest.Repository, j.manifest.Branch}, "-"))
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

func (j *Jenkins) ShowConsoleOutput() {
	var lastChar int
	for {
		build, err := j.getJob().GetLastBuild()
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

func (j *Jenkins) OpenPage(p ...string) error {
	base := []string{j.cfg.Jenkins.Server, "job/BenchLabs/job", j.manifest.Repository, "job", j.manifest.Branch}
	return utils.OpenURI(append(base, p...)...)
}

func (j *Jenkins) BuildJob(async bool, force bool) {
	jobName := j.getJobName()
	job := j.getJob()
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
		newBuild, err := j.getJob().GetLastBuild()
		if err == nil && (lastBuild == nil || (lastBuild.GetUrl() != newBuild.GetUrl())) {
			os.Stderr.WriteString("\n")
			break
		} else if err != nil && err.Error() != "404" {
			log.Fatalf("Failed to get build status: %v", err)
		}
		os.Stderr.WriteString(".")
		time.Sleep(2 * time.Second)
	}
	j.ShowConsoleOutput()
}

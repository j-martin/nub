package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"log"
	"os/exec"
)

func main() {
	//TODO: Use proper subcommands, e.g. bub open jenkins
	// with something like https://github.com/urfave/cli#subcommands

	//TODO: Add open in splunk staging and prod
	//TODO: Add open datadog
	//TODO: Add other useful things.

	update := flag.Bool("update", false, "Update the state current repo.")
	list := flag.Bool("list", false, "List all projects.")
	validate := flag.Bool("validate", false, "Validate manifest.")

	repo := flag.Bool("repo", false, "Open repo.")
	issues := flag.Bool("issues", false, "Open issues.")
	pr := flag.Bool("pr", false, "Open pr.")
	branches := flag.Bool("branches", false, "Open branches.")
	raml := flag.Bool("raml", false, "Open raml.")

	jenkins := flag.Bool("jenkins", false, "Open jenkins.")
	jenkinsConsole := flag.Bool("jenkins-console", false, "Open jenkins console.")

	wiki := flag.Bool("docs", false, "Open wiki.")
	circle := flag.Bool("circle", false, "Open circle.")

	flag.Parse()
	m := BuildManifest()

	if *validate {
		//TODO: Build proper validation
		yml, _ := yaml.Marshal(m)
		log.Println(string(yml))
	}

	if *update {
		//TODO: Allow to pass version at build time so
		// we can list the docker and eb images version available.
		StoreManifest(m)
	}

	if *list {
		manifests := GetAllManifests()
		yml, _ := yaml.Marshal(manifests)
		log.Println(string(yml))
	}

	ghUrl := "https://github.com/BenchLabs/"
	jenkinsUrl := "https://jenkins.example.com/jobs/BenchLabs/job/"
	wikiUrl := "https://example.atlassian.net/wiki/display/dev/"
	circleUrl := "https://circleci.com/gh/BenchLabs/"

	if *repo {
		exec.Command("open", ghUrl + m.Repository).Run()
	}

	if *issues {
		exec.Command("open", ghUrl + m.Repository + "/issues").Run()
	}

	if *branches {
		exec.Command("open", ghUrl + m.Repository + "/branches").Run()
	}

	if *pr {
		exec.Command("open", ghUrl + m.Repository + "/pulls").Run()
	}

	if *raml {
		exec.Command("open", ghUrl + m.Repository + "/pulls").Run()
	}

	if *jenkins {
		exec.Command("open", jenkinsUrl + m.Repository).Run()
	}

	if *jenkinsConsole {
		exec.Command("open", jenkinsUrl + m.Repository + "/lastBuild/console").Run()
	}

	if *wiki {
		exec.Command("open", wikiUrl + m.Name).Run()
	}

	if *circle {
		exec.Command("open", circleUrl + m.Repository).Run()
	}
}

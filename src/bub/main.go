package main

import (
	"github.com/docopt/docopt-go"
	"gopkg.in/yaml.v2"
	"log"
)

func main() {
	//TODO: Add open in splunk staging and prod
	//TODO: Add open datadog
	usage := `bub.

Usage:
  bub list
  bub manifest | m update --service-version <value>
  bub manifest | m validate
  bub open | o repo
  bub open | o issues
  bub open | o pr
  bub open | o branches
  bub open | o raml
  bub open | o jenkins
  bub open | o jenkins console
  bub open | o docs
  bub open | o circle
  bub -h | --help
  bub --version

Options:
  -h --help             Show this screen.
  --service-version     Show version.
  --version             Version of the service to update.`

	arguments, _ := docopt.Parse(usage, nil, true, "bub 0.1-experimental", false)

	m := BuildManifest()

	if arguments["validate"].(bool) {
		//TODO: Build proper validation
		yml, _ := yaml.Marshal(m)
		log.Println(string(yml))
	}

	if arguments["update"].(bool) {
		//TODO: Allow to pass version at build time so
		// we can list the docker and eb images version available.
		StoreManifest(m)
	}

	if arguments["list"].(bool) {
		manifests := GetAllManifests()
		yml, _ := yaml.Marshal(manifests)
		log.Println(string(yml))
	}

	if arguments["repo"].(bool) {
		OpenGH(m, "")
	}

	if arguments["issues"].(bool) {
		OpenGH(m, "issues")
	}

	if arguments["branches"].(bool) {
		OpenGH(m, "branches")
	}

	if arguments["pr"].(bool) {
		OpenGH(m, "pulls")
	}

	if arguments["raml"].(bool) {
		url := "https://github.com/BenchLabs/bench-raml/tree/master/specs/"
		OpenURI(url + m.Repository + ".raml")
	}

	if arguments["jenkins"].(bool) {
		OpenJenkins(m, "")
	}

	if arguments["console"].(bool) {
		OpenJenkins(m, "lastBuild/console")
	}

	if arguments["docs"].(bool) {
		wikiUrl := "https://example.atlassian.net/wiki/display/dev/"
		OpenURI(wikiUrl + m.Name)
	}

	if arguments["circle"].(bool) {
		circleUrl := "https://circleci.com/gh/BenchLabs/"
		OpenURI(circleUrl + m.Repository)
	}
}

package main

import (
	"github.com/docopt/docopt-go"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

func main() {
	//TODO: Add open in splunk staging and prod
	//TODO: Add open datadog
	usage := `bub.

Usage:
  bub list
  bub repository | r sync [--force]
  bub manifest | m update [--artifact-version <value>]
  bub manifest | m validate
  bub open | o repo
  bub open | o issues
  bub open | o pr
  bub open | o branches
  bub open | o compare
  bub open | o raml
  bub open | o jenkins
  bub open | o jenkins console
  bub open | o jenkins trigger
  bub open | o splunk
  bub open | o splunk staging
  bub open | o docs
  bub open | o circle
  bub open | o circle branch
  bub -h | --help
  bub --version

Options:
  -h --help                    Show this screen.
  --artifact-version <value>   Artifact version [default: n/a].
  --force                      Force sync, wihtout prompt.
  --version                    Version of the service to update.`

	args, _ := docopt.Parse(usage, nil, true, "bub 0.1-experimental", false)

	if args["list"].(bool) {
		manifests := GetAllManifests()
		yml, _ := yaml.Marshal(manifests)
		log.Println(string(yml))
		os.Exit(0)

	} else if args["sync"].(bool) {
		msg := "Clone and/or Update all Bench repositories?\n" +
			"Existing work will be stashed and pull the master branch.\n" +
			"Please make sure you in the directory where you " +
			"store your repos and not a specific repo."

		if args["--force"].(bool) || askForConfirmation(msg) {
			SyncRepositories(GetAllManifests())
		}
		os.Exit(0)
	}

	m := BuildManifest(args["--artifact-version"].(string))

	if args["validate"].(bool) {
		//TODO: Build proper validation
		yml, _ := yaml.Marshal(m)
		log.Println(string(yml))

	} else if args["update"].(bool) {
		StoreManifest(m)

	} else if args["repo"].(bool) {
		OpenGH(m, "")

	} else if args["issues"].(bool) {
		OpenGH(m, "issues")

	} else if args["branches"].(bool) {
		OpenGH(m, "branches")

	} else if args["pr"].(bool) {
		OpenGH(m, "pulls")

	} else if args["raml"].(bool) {
		OpenURI("https://github.com/BenchLabs/bench-raml/tree/master/specs/" + m.Repository + ".raml")

	} else if args["jenkins"].(bool) && args["console"].(bool) {
		OpenJenkins(m, "lastBuild/console")

	} else if args["jenkins"].(bool) && args["trigger"].(bool) {
		OpenJenkins(m, "job/master/trigger")

	} else if args["jenkins"].(bool) {
		OpenJenkins(m, "")

	} else if args["splunk"].(bool) {
		OpenSplunk(m, args["staging"].(bool))

	} else if args["docs"].(bool) {
		OpenURI("https://example.atlassian.net/wiki/display/dev/" + m.Name)

	} else if args["circle"].(bool) {
		OpenCircle(m, args["branch"].(bool))
	}
}

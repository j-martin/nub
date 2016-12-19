package main

import (
	"errors"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const manifestFile = ".bench.yml"

type Manifest struct {
	Name         string
	Active       bool
	Repository   string
	LastUpdate   int64
	Language     string
	Types        []string
	Dependencies []Dependency
	Protocols    []Protocol
	Version      string
	Branch       string
	Readme       string
	ChangeLog    string
	Page         string
}

type Dependency struct {
	Name    string
	Version string
}

type Protocol struct {
	Type string
	Path string
}

func LoadManifest(version string) (Manifest, error) {
	m := Manifest{}

	if !IsInRepository() {
		return Manifest{}, errors.New("Must be executed in a repository.")
	}

	data, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		return Manifest{}, errors.New("Must be executed in a repository.")
	}

	readme, err := ioutil.ReadFile("README.md")
	if err != nil {
		return Manifest{}, err
	}

	changelog, err := ioutil.ReadFile("CHANGELOG.md")

	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return Manifest{}, err
	}

	m.LastUpdate = time.Now().Unix()
	m.Repository = GetCurrentRepositoryName()
	m.Branch = GetCurrentBranch()
	m.Version = version

	m.Readme = string(readme)
	m.ChangeLog = string(changelog)

	return m, nil
}

func CreateManifest() {

	manifest := Manifest{
		Name: GetCurrentRepositoryName(),
	}
	manifestString := `---
name: {{.Name}}
active: true
language: scala
types:
  - service
dependencies:
  - name: activemq
	version: 5.13
  - name: postgres
	version: 9.4
protocols:
  - type: raml
	path: client/src/main/raml
page: pageID from confluence, not the name.
`
	manifestTemplate, err := template.New("manifest").Parse(manifestString)
	if err != nil {
		panic(err)
	}

	fileExists, err := exists(manifestFile)
	if err != nil {
		log.Fatal(err)
	}

	if !fileExists {
		log.Println("Creating manifest.")
		writer, err := os.Create(manifestFile)
		if err != nil {
			log.Fatal(err)
		}
		manifestTemplate.Execute(writer, manifest)
	}

	log.Println("Edit the manifest file.")
	editFile(manifestFile)
	log.Println("Done. Don't forget to add and commit the file.")
}

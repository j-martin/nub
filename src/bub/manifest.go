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
	Platform     string // what is it running on
	Language     string
	Languages    []string
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
	// name of the dependency
	Name string
	// optional, explicit name of the dependency.
	// if not defined the name will be <service>-<dependencyName> e.g. mainapp-mysql
	UniqueName string
	// e.g. postgres 9.6
	Version string
	// e.g. why it depends on it
	Description string
	// service, database, front-end
	Type string
	// not managed / controlled by us on AWS
	Dedicated bool
	// not managed / controlled by us on AWS
	External bool
	// e.g. most services don't communicate directly with a service.
	// like the service relies on it on putting a message/event in the broadcast queue
	Implicit bool
	// to (default), from, both
	Direction string
}

type Protocol struct {
	Type string
	Path string
}

type Manifests []Manifest

func (e Manifests) Len() int {
	return len(e)
}

func (e Manifests) Less(i, j int) bool {
	return e[i].Name < e[j].Name
}

func (e Manifests) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func LoadManifest(version string) (Manifest, error) {
	m := Manifest{}

	if !IsInRepository() {
		return Manifest{}, errors.New("must be executed in a repository.")
	}

	data, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		data, err = ioutil.ReadFile("manifest.yml")
	}
	err = yaml.Unmarshal(data, &m)

	if len(m.Languages) == 0 && m.Language != "" {
		m.Languages = []string{m.Language}
	}

	if m.Language == "" && len(m.Languages) > 0 {
		m.Language = m.Languages[0]
	}

	m.LastUpdate = time.Now().Unix()
	m.Repository = GetCurrentRepositoryName()
	m.Branch = GetCurrentBranch()
	m.Version = version

	readme, _ := ioutil.ReadFile("README.md")
	m.Readme = string(readme)

	changelog, _ := ioutil.ReadFile("CHANGELOG.md")
	m.ChangeLog = string(changelog)

	return m, err
}

func CreateManifest() {

	manifest := Manifest{
		Name: GetCurrentRepositoryName(),
	}
	manifestString := `---
name: {{.Name}}
active: true
languages:
  - scala
types:
  - service
dependencies:
  - name: activemq
    direction: both
  - name: postgres
    version: 9.6
protocols:
  - type: raml
    path: client/src/main/raml
page: pageID from confluence, not the name.
`
	manifestTemplate, err := template.New("manifest").Parse(manifestString)
	if err != nil {
		panic(err)
	}

	fileExists, err := pathExists(manifestFile)
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

func IsType(m Manifest, manifestType string) bool {
	for _, i := range m.Types {
		if i == manifestType {
			return true
		}
	}
	return false
}


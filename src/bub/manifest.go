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

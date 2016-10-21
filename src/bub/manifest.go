package main

import (
	"io/ioutil"
	"log"
	"time"
	"gopkg.in/yaml.v2"
)

type Manifest struct {
	Name         string
	Active       bool
	Repository   string
	LastUpdate   int64
	Language     string
	Types        []string
	Dependencies []Dependency
	Protocols    []Protocol
	Readme       string
	ChangeLog    string
}

type Dependency struct {
	Name    string
	Version string
}

type Protocol struct {
	Type string
	Path string
}

func BuildManifest() Manifest {
	m := Manifest{}

	data, err := ioutil.ReadFile(".bench.yml")
	if err != nil {
		log.Fatalf("Could not %v", err)
	}

	readme, err := ioutil.ReadFile("README.md")
	if err != nil {
		log.Printf("Could not %v", err)
	}

	changelog, err := ioutil.ReadFile("CHANGELOG.md")
	if err != nil {
		log.Printf("Could not %v", err)
	}

	err = yaml.Unmarshal(data, &m)
	if err != nil {
		log.Fatalf("Could not unmarshal manifest: %v", err)
	}

	m.LastUpdate = time.Now().Unix()
	m.Repository = GetCurrentRepositoryName()

	m.Readme = string(readme)
	m.ChangeLog = string(changelog)

	return m
}


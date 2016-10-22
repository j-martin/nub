package main

import (
	"log"
	"os/exec"
)

func OpenURI(uri string) {
	log.Printf("Opening: %v", uri)
	exec.Command("open", uri).Run()
}

func OpenGH(m Manifest, p string) {
	url := "https://github.com/BenchLabs/"
	OpenURI(url + m.Repository + "/" + p)
}

func OpenJenkins(m Manifest, p string) {
	url := "https://jenkins.example.com/jobs/BenchLabs/job/"
	OpenURI(url + m.Repository + "/" + p)

}

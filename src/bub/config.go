package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
	"os/user"
	"log"
)

type Configuration struct {
	Github     struct {
				   Organization string
			   }
	Jenkins    struct {
				   Server, Username, Password string
			   }
	Confluence struct {
				   Server, Username, Password string
			   }
}

func LoadConfiguration() Configuration {
	cfg := Configuration{}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	configDir := path.Join(usr.HomeDir, ".config", "bub")
	configPath := path.Join(configDir, "config.yml")

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("Could not parse yaml file. %v", err)
	}

	return cfg
}

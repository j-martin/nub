package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
)

type RDSConfiguration struct {
	Prefix, Database, User, Password string
}

type Environment struct {
	Name, Domain, Jumphost string
}
type Configuration struct {
	AWS struct {
		Regions      []string
		RDS          []RDSConfiguration
		Environments []Environment
	}
	Github struct {
		Organization string
	}
	Jenkins struct {
		Server, Username, Password string
	}
	Confluence struct {
		Server, Username, Password string
	}
	Circle struct {
		Token string
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
		log.Print("No bub configuration found. Please run `bub setup`")
		return cfg
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Printf("Could not parse yaml file. %v", err)
		return cfg
	}

	if len(cfg.AWS.Regions) == 0 {
		cfg.AWS.Regions = []string{"us-east-1", "us-west-2"}
	}

	return cfg
}

func EditConfig() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configPath := path.Join(usr.HomeDir, ".config", "bub", "config.yml")
	editFile(configPath)
}

func Setup() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	//TODO: move aws configuration to config.yml
	awsCredentials := `[default]
output=json
region=us-east-1
aws_access_key_id = CHANGE_ME
aws_secret_access_key = CHANGE_ME`

	config := `---
aws:
  regions:
    - us-east-1
    - us-west-2

  rds:
  # the first prefix match will be used. The database will be infered from the host name.
    - prefix: staging
      database: <optional>
      user: <optional>
      password: <optional>

  environments:
    - name: prod
      domain: example.com
      jumphost: jump.example.com
    - name: staging2
      domain: staging2.example.com
      jumphost: jump.staging2.example.com
    - name: staging
      domain: example.com
      jumphost: jump.example.com

github:
  organization: benchlabs

jenkins:
  server: "https://jenkins.example.com"
  username: <optional-change-me>
  password: <optional-change-me>

confluence:
  server: "https://example.atlassian.net/wiki"
  username: <optional-change-me>
  password: <optional-change-me>

circle:
  token: <optional-change-me>
`

	createDir(path.Join(usr.HomeDir, ".aws"), "credentials", awsCredentials)
	createDir(path.Join(usr.HomeDir, ".config", "bub"), "config.yml", config)

	log.Println("Done.")
}

func createDir(directory string, filename string, content string) {
	filePath := path.Join(directory, filename)
	dirExists, err := exists(directory)
	if err != nil {
		log.Fatal(err)
	}

	if !dirExists {
		os.MkdirAll(directory, 0700)
	}

	fileExists, err := exists(filePath)
	if err != nil {
		log.Fatal(err)
	}

	if !fileExists {
		log.Printf("Creating %s file.", filename)
		ioutil.WriteFile(filePath, []byte(content), 0700)
	}

	log.Printf("Editing %s.", filename)
	editFile(filePath)

}

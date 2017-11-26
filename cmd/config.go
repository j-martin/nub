package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"

	"gopkg.in/yaml.v2"
)

type RDSConfiguration struct {
	Prefix, Database, User, Password string
}

type Environment struct {
	Prefix, Jumphost, Region string
}

type User struct {
	Name, Slack string
}

type Configuration struct {
	AWS struct {
		Regions      []string
		RDS          []RDSConfiguration
		Environments []Environment
	}
	GitHub struct {
		Organization string
	}
	Users []User
	JIRA  struct {
		Server string
	}
	Jenkins struct {
		Server, Username, Password string
	}
	Splunk struct {
		Server string
	}
	Confluence struct {
		Server, Username, Password string
	}
	Circle struct {
		Token string
	}
	Updates struct {
		Region, Bucket, Prefix string
	}
	Ssh struct {
		ConnectTimeout uint `yaml:"connectTimeout"`
	}
}

var config string = `---
aws:
	regions:
		- us-east-1
		- us-west-2

	rds:
		# The first prefix match will be used.
		# The database name, unless specified, will be infered from the host name.
		- prefix: staging
			database: <optional>
			user: <optional>
			password: <optional>

	environments:
		- prefix: staging2
			jumphost: jump.staging2.example.com
			region: us-west-2
		- prefix: staging
			jumphost: jump.example.com
			region: us-west-2
		# if not prefix, act as a catch all.
		- jumphost: jump.example.com
			region: us-east-1

github:
	organization: benchlabs

jenkins:
	server: "https://jenkins.example.com"
	username: <optional-change-me>
	password: <optional-api-token-also-works>

confluence:
	server: "https://example.atlassian.net/wiki"
	username: <optional-change-me>
	password: <optional-change-me>

splunk:
	server: "https://splunk.example.com"

circle:
	token: <optional-change-me>

updates:
	region: us-east-1
	bucket: s3bucket
	prefix: contrib/bub

ssh:
	connectTimeout: 3
`

func loadConfiguration() Configuration {
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

func editConfiguration() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configPath := path.Join(usr.HomeDir, ".config", "bub", "config.yml")
	editFile(configPath)
}

func setup() {
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

	createDir(path.Join(usr.HomeDir, ".aws"), "credentials", awsCredentials)
	createDir(path.Join(usr.HomeDir, ".config", "bub"), "config.yml", config)

	log.Println("Done.")
}

func createDir(directory string, filename string, content string) {
	filePath := path.Join(directory, filename)
	dirExists, err := pathExists(directory)
	if err != nil {
		log.Fatal(err)
	}

	if !dirExists {
		os.MkdirAll(directory, 0700)
	}

	fileExists, err := pathExists(filePath)
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

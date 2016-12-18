package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
)

var config = aws.Config{Region: aws.String("us-east-1")}

func Setup() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	credentials := `[default]
output=json
region=us-east-1
aws_access_key_id = CHANGE_ME
aws_secret_access_key = CHANHE_ME`

	credentialsPath := path.Join(usr.HomeDir, ".aws", "credentials")
	credentialsDir := path.Base(credentialsPath)

	dirExists, err := exists(credentialsDir)
	if err != nil {
		log.Fatal(err)
	}

	if !dirExists {
		os.MkdirAll(credentialsDir, 0700)
	}

	fileExists, err := exists(credentialsPath)
	if err != nil {
		log.Fatal(err)
	}

	if !fileExists {
		log.Println("Creating credential file.")
		ioutil.WriteFile(credentialsPath, []byte(credentials), 0700)
	}

	log.Println("Edit your credentials.")
	editFile(credentialsPath)
	log.Println("Done.")
}

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/benchlabs/bub/utils"
	"log"
	"os/user"
	"path"
)

func GetAWSConfig(region string) aws.Config {
	return aws.Config{Region: aws.String(region)}
}

func MustSetupConfig() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	utils.Prompt("You will have to enter your AWS credentials next. Ask an AWS Admin for your credentials. Continue?")

	awsCredentials := `[default]
output=json
region=us-east-1
aws_access_key_id = CHANGE_ME
aws_secret_access_key = CHANGE_ME`

	utils.CreateAndEdit(path.Join(usr.HomeDir, ".aws", "credentials"), awsCredentials)
}

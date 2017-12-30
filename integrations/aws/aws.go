package aws

import (
	"github.com/aws/aws-sdk-go/aws"
)

func GetAWSConfig(region string) aws.Config {
	return aws.Config{Region: aws.String(region)}
}

package main

import (
	"github.com/aws/aws-sdk-go/aws"
)

func getAwsConfig(region string) aws.Config {
	return aws.Config{Region: aws.String(region)}
}

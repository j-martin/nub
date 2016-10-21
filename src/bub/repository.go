package main

import (
	"log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

//TODO: Add GetAllActive

var table = aws.String("manifests")

func GetAllManifests() []Manifest {
	manifests := []Manifest{}
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))
	params := &dynamodb.ScanInput{TableName: table}
	result, err := svc.Scan(params)

	if err != nil {
		log.Fatal(err)
	}

	dynamodbattribute.UnmarshalListOfMaps(result.Items, &manifests)
	return manifests
}

func StoreManifest(m Manifest) {
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))
	manifest, err := dynamodbattribute.MarshalMap(m)

	if err != nil {
		log.Println(err)
	}

	params := &dynamodb.PutItemInput{TableName: table, Item: manifest}
	_, err = svc.PutItem(params)

	if err != nil {
		log.Println(err)
	}

	log.Printf("%v updated.", m.Name)
}

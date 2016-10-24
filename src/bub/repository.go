package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
)

//TODO: Add GetAllActive

var table = aws.String("manifests")
var config = aws.Config{Region: aws.String("us-east-1")}

func GetAllManifests() []Manifest {
	log.Println("Fetching all manifests.")
	manifests := []Manifest{}
	svc := dynamodb.New(session.New(&config))
	params := &dynamodb.ScanInput{TableName: table}
	result, err := svc.Scan(params)

	if err != nil {
		log.Fatal(err)
	}

	dynamodbattribute.UnmarshalListOfMaps(result.Items, &manifests)
	return manifests
}

func StoreManifest(m Manifest) {
	log.Printf("Updating manifest: %v", m.Name)
	svc := dynamodb.New(session.New(&config))
	manifest, err := dynamodbattribute.MarshalMap(m)

	if err != nil {
		log.Println(err)
	}

	params := &dynamodb.PutItemInput{TableName: table, Item: manifest}
	_, err = svc.PutItem(params)

	if err != nil {
		log.Println(err)
	}

	log.Println("Updating manifest: complete.")
}

package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
)

var manifestsTable = aws.String("manifests")

func getDynamoSvc() *dynamodb.DynamoDB {
	return dynamodb.New(session.New(&config))
}

func GetAllActiveManifests() []Manifest {
	ms := []Manifest{}
	for _, m := range GetAllManifests() {
		if m.Active {
			ms = append(ms, m)
		}
	}
	return ms
}

func GetAllManifests() []Manifest {
	log.Println("Fetching all manifests.")
	manifests := []Manifest{}
	params := &dynamodb.ScanInput{TableName: manifestsTable}
	result, err := getDynamoSvc().Scan(params)

	if err != nil {
		log.Fatal(err)
	}

	dynamodbattribute.UnmarshalListOfMaps(result.Items, &manifests)
	return manifests
}

func StoreManifest(m Manifest) {
	log.Printf("Updating manifest: %v", m.Name)
	manifest, err := dynamodbattribute.MarshalMap(m)

	if err != nil {
		log.Println(err)
	}

	params := &dynamodb.PutItemInput{TableName: manifestsTable, Item: manifest}
	_, err = getDynamoSvc().PutItem(params)

	if err != nil {
		log.Println(err)
	}

	log.Println("Updating manifest: complete.")
}

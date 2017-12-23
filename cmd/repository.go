package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"sort"
)

type manifestRepository struct {
	db *dynamodb.DynamoDB
}

var manifestsTable = aws.String("manifests")

func GetManifestRepository() *manifestRepository {
	config := getAWSConfig("us-east-1")
	return &manifestRepository{db: dynamodb.New(session.New(&config))}
}

func (r *manifestRepository) GetAllActiveManifests() []Manifest {
	ms := Manifests{}
	for _, m := range r.GetAllManifests() {
		if m.Active {
			ms = append(ms, m)
		}
	}
	return ms
}

func (r *manifestRepository) GetAllManifests() []Manifest {
	log.Println("Fetching all manifests.")
	manifests := Manifests{}
	params := &dynamodb.ScanInput{TableName: manifestsTable}
	result, err := r.db.Scan(params)

	if err != nil {
		log.Fatal(err)
	}

	dynamodbattribute.UnmarshalListOfMaps(result.Items, &manifests)
	sort.Sort(manifests)
	return manifests
}

func (r *manifestRepository) StoreManifest(m Manifest) {
	log.Printf("Updating manifest: %v", m.Name)
	manifest, err := dynamodbattribute.MarshalMap(m)

	if err != nil {
		log.Println(err)
	}

	params := &dynamodb.PutItemInput{TableName: manifestsTable, Item: manifest}
	_, err = r.db.PutItem(params)

	if err != nil {
		log.Println(err)
	}

	log.Println("Updating manifest: complete.")
}

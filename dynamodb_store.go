package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ddbClient *dynamodb.Client

const tableName = "PDFMetadata"

func initDynamo() {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           "http://dynamodb:8000",
			SigningRegion: "us-east-1",
		}, nil
	})
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")),
	)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
	var lastErr error
	for i := 0; i < 5; i++ {
		_, err = ddbClient.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
			TableName: aws.String(tableName),
			AttributeDefinitions: []types.AttributeDefinition{
				{AttributeName: aws.String("filename"), AttributeType: types.ScalarAttributeTypeS},
			},
			KeySchema: []types.KeySchemaElement{
				{AttributeName: aws.String("filename"), KeyType: types.KeyTypeHash},
			},
			BillingMode: types.BillingModePayPerRequest,
		})
		if err == nil {
			return
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("Failed to create DynamoDB table after retries: %v", lastErr)
}

func storePDFMeta(filename string, fieldCount int) {
	_, err := ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"filename":   &types.AttributeValueMemberS{Value: filename},
			"uploadedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
			"fields":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", fieldCount)},
		},
	})
	if err != nil {
		log.Println("Failed to store metadata:", err)
	}
}

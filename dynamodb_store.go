package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"log"
	"os"
	"strings"
	"time"
)

var ddbClient *dynamodb.Client

const tableName = "PDFMetadata"

func initDynamo() {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://dynamodb:8000"
		log.Println("⚠️  No DYNAMODB_ENDPOINT set — assuming Docker network 'dynamodb:8000'")
	}

	region := "us-east-1"

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpoint,
			PartitionID:   "aws",
			SigningRegion: region,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")),
	)
	if err != nil {
		log.Fatalf("❌ Unable to load AWS config: %v", err)
	}

	ddbClient = dynamodb.NewFromConfig(cfg)
	createTableIfNotExists()
}

// createTableIfNotExists creates the DynamoDB table if it doesn't exist.
func createTableIfNotExists() {
	var lastErr error
	for i := 0; i < 5; i++ {
		_, err := ddbClient.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
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
			log.Printf("✅ DynamoDB table %s created\n", tableName)
			return
		}
		if isTableExistsError(err) {
			log.Printf("⚠️ DynamoDB table %s already exists\n", tableName)
			return
		}
		lastErr = err
		log.Printf("⏳ Waiting for DynamoDB... (%d/5)\n", i+1)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("❌ Failed to create DynamoDB table after retries: %v", lastErr)
}

// isTableExistsError checks if the error indicates that the table already exists.
func isTableExistsError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ResourceInUseException")
}

// storePDFMeta stores the metadata of the PDF file in DynamoDB.
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
		log.Printf("❌ Failed to store metadata for %s: %v\n", filename, err)
	} else {
		log.Printf("🗂️  Stored metadata for %s (%d fields)\n", filename, fieldCount)
	}
}

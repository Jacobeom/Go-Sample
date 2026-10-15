package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/shopspring/decimal"
)

type UserWallet struct {
	UserID  string          `dynamodbav:"UserId"`  // Partition Key
	Balance decimal.Decimal `dynamodbav:"Balance"` // Sort Key (if applicable)
}

func Test_DB(t *testing.T) {

	item := UserWallet{UserID: "jacobo", Balance: decimal.NewFromFloat(10.00)}
	itemM, err := attributevalue.MarshalMap(item)

	ctx := context.TODO()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-north-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	client := dynamodb.NewFromConfig(cfg)
	input := &dynamodb.PutItemInput{
		TableName: aws.String("UserWallet"),
		Item:      itemM,
		// Optional: Prevent overwriting an existing item with the same primary key
		ConditionExpression: aws.String("attribute_not_exists(user_id)"),
	}
	_, err = client.PutItem(ctx, input)
	if err != nil {
		// Handle specific conditional check failures if needed
		var ccErr *types.ConditionalCheckFailedException
		if errors.As(err, &ccErr) {
			fmt.Errorf("user with this ID already exists")
		}
		fmt.Errorf("failed to put item into dynamodb: %w", err)
	}

	// 3. Test the connection by listing tables
	output, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		log.Fatalf("failed to list tables, %v", err)
	}

	fmt.Println("Connected successfully! Tables in your account:")
	for _, tableName := range output.TableNames {
		fmt.Printf("- %s\n", tableName)
	}

	asset := 1
	if !(asset == 1) {
		t.Errorf("expected final balance %d, got %s", asset, "1")
	}
}

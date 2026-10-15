package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/shopspring/decimal"
)

// DynamoAccountStore implements models.AccountStore using AWS DynamoDB.
type DynamoAccountStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoAccountStore creates a new instance of DynamoAccountStore.
func NewDynamoAccountStore(client *dynamodb.Client, tableName string) *DynamoAccountStore {
	return &DynamoAccountStore{
		client:    client,
		tableName: tableName,
	}
}

// userWalletDB is the internal persistence model for DynamoDB storage.
// Storing the Balance as a string avoids any floating-point precision loss.
type UserWalletDB struct {
	UserID  string  `dynamodbav:"UserId"`  // Partition Key
	Balance float64 `dynamodbav:"Balance"` // Sort Key (if applicable)
}

// Balance retrieves the user's current balance from DynamoDB.
func (s *DynamoAccountStore) Balance(ctx context.Context, userId string) (*BalanceResponse, error) {

	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("UserId = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userId},
		},
		Limit: aws.Int32(1),
	}
	output, err := s.client.Query(ctx, input)

	if err != nil {
		return nil, fmt.Errorf("failed to query wallet: %w", err)
	}

	if len(output.Items) == 0 {
		return nil, errors.New("wallet not found")
	}

	var wallet UserWalletDB
	err = attributevalue.UnmarshalMap(output.Items[0], &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet: %w", err)
	}

	return &BalanceResponse{UserId: userId, Balance: decimal.NewFromFloat(wallet.Balance), Message: "Balance successful"}, nil
}

func (s *DynamoAccountStore) Deposit(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("deposit amount must be greater than zero")
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserId": &types.AttributeValueMemberS{Value: userId},
		},
		// Adds to existing Balance or initializes it to 0 if the item doesn't exist yet
		UpdateExpression: aws.String("SET Balance = if_not_exists(Balance, :zero) + :amount"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":amount": &types.AttributeValueMemberN{Value: amount.String()},
			":zero":   &types.AttributeValueMemberN{Value: "0"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	}

	output, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute atomic deposit: %w", err)
	}

	var wallet UserWalletDB
	err = attributevalue.UnmarshalMap(output.Attributes, &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated wallet: %w", err)
	}

	return &BalanceResponse{
		UserId:  userId,
		Balance: decimal.NewFromFloat(wallet.Balance),
		Message: "Deposit successful",
	}, nil

}

// Withdraw subtracts an amount from the user's balance with insufficient funds validation.
func (s *DynamoAccountStore) Withdraw(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("withdrawal amount must be greater than zero")
	}

	// DynamoDB atomic update with a condition ensuring the balance won't drop below zero
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserId": &types.AttributeValueMemberS{Value: userId},
		},
		// Subtracts the amount only if the balance exists and is greater than or equal to the withdrawal amount
		UpdateExpression:    aws.String("SET Balance = Balance - :amount"),
		ConditionExpression: aws.String("attribute_exists(Balance) AND Balance >= :amount"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":amount": &types.AttributeValueMemberN{Value: amount.String()},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	}

	output, err := s.client.UpdateItem(ctx, input)
	if err != nil {
		// Handle conditional check failure specifically for insufficient funds or missing account
		var cerr *types.ConditionalCheckFailedException
		if errors.As(err, &cerr) {
			return nil, fmt.Errorf("insufficient funds or account not found")
		}
		return nil, fmt.Errorf("failed to execute atomic withdrawal: %w", err)
	}

	// Parse the newly updated balance returned from DynamoDB
	var newBalance decimal.Decimal
	if output.Attributes != nil {
		if balAttr, ok := output.Attributes["Balance"].(*types.AttributeValueMemberN); ok {
			var parseErr error
			newBalance, parseErr = decimal.NewFromString(balAttr.Value)
			if parseErr != nil {
				return nil, fmt.Errorf("failed to parse updated balance: %w", parseErr)
			}
		}
	}

	return &BalanceResponse{
		UserId:  userId,
		Balance: newBalance,
	}, nil
}

package main

import (
	"context"
	"fmt"
	"log"
	"main/auth"
	_ "main/docs"
	"main/internal"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Player Balance API
// @version         1.0
// @description     API server for player deposit and withdrawal operations.
// @host            localhost:8080
// @BasePath        /

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-north-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		tableName = "UserWallet" // fallback default table name
	}
	var store internal.AccountStore = internal.NewDynamoAccountStore(dynamoClient, tableName)

	server := internal.NewServer(store)
	authService := auth.NewAuthService(dynamoClient, "Login")

	http.HandleFunc("/register", authService.RegisterLogin)
	http.HandleFunc("/login", authService.Login)
	http.HandleFunc("/logout", auth.AuthMiddleware(authService.Logout))
	http.HandleFunc("/deposit", auth.AuthMiddleware(server.HandleDeposit))
	http.HandleFunc("/withdraw", auth.AuthMiddleware(server.HandleWithdraw))
	http.HandleFunc("/balance", auth.AuthMiddleware(server.HandleGetBalance))
	http.HandleFunc("/swagger/", auth.AuthMiddleware(httpSwagger.WrapHandler))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

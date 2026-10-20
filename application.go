package main

import (
	"context"
	"fmt"
	"log"
	"main/auth"
	_ "main/docs"
	"main/internal"
	"net/http"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	httpSwagger "github.com/swaggo/http-swagger"
)

type Config struct {
	Server   ServerConfig `toml:"server"`
	Database AWSConfig    `toml:"awsconfig"`
}

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type AWSConfig struct {
	Region      string `toml:"region"`
	WalletTable string `toml:"wallet_table"`
	LoginTable  string `toml:"login_table"`
}

// @title           Player Balance API
// @version         1.0
// @description     API server for player deposit and withdrawal operations.
// @host            localhost:8080
// @BasePath        /

func main() {
	var appConfig Config
	ctx := context.Background()

	_, err := toml.DecodeFile("config.toml", &appConfig)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(appConfig.Database.Region))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	var store internal.AccountStore = internal.NewDynamoAccountStore(dynamoClient, appConfig.Database.WalletTable)

	server := internal.NewServer(store)
	authService := auth.NewAuthService(dynamoClient, appConfig.Database.LoginTable)

	http.HandleFunc("/register", authService.RegisterLogin)
	http.HandleFunc("/login", authService.Login)
	http.HandleFunc("/logout", auth.AuthMiddleware(authService.Logout))
	http.HandleFunc("/deposit", auth.AuthMiddleware(server.HandleDeposit))
	http.HandleFunc("/withdraw", auth.AuthMiddleware(server.HandleWithdraw))
	http.HandleFunc("/balance", auth.AuthMiddleware(server.HandleGetBalance))
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)
	hostAndPort := fmt.Sprintf("%s:%d", appConfig.Server.Host, appConfig.Server.Port)

	fmt.Printf("Server starting on http://%s\n", hostAndPort)
	if err := http.ListenAndServe(":"+fmt.Sprintf("%d", appConfig.Server.Port), nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

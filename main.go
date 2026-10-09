package main

import (
	"fmt"
	"log"
	_ "main/docs"
	"main/models"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Player Balance API
// @version         1.0
// @description     API server for player deposit and withdrawal operations.
// @host            localhost:8080
// @BasePath        /

func main() {
	store := models.NewAccountStore()

	http.HandleFunc("/deposit", store.HandleDeposit)
	http.HandleFunc("/withdraw", store.HandleWithdraw)
	http.HandleFunc("/balance", store.HandleGetBalance)
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	port := ":5000"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

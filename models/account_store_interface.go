package models

import (
	"github.com/shopspring/decimal"
)

// ==========================================
// 1. THE INTERFACE
// ==========================================

type AccountStore interface {
	Deposit(userId string, amount decimal.Decimal) (*BalanceResponse, error)
	Withdraw(userId string, amount decimal.Decimal) (*BalanceResponse, error)
	Balance(userId string) (*BalanceResponse, error)
}

// ==========================================
// SHARED PAYLOADS & RESPONSES
// ==========================================

type TransactionRequest struct {
	UserId string          `json:"user_id"`
	Amount decimal.Decimal `json:"amount"`
}

type BalanceResponse struct {
	UserId  string          `json:"player_id"`
	Balance decimal.Decimal `json:"balance"`
	Message string          `json:"message,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

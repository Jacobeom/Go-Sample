package models

import (
	"context"

	"github.com/shopspring/decimal"
)

// ==========================================
// 1. THE INTERFACE
// ==========================================

type AccountStore interface {
	Deposit(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error)
	Withdraw(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error)
	Balance(ctx context.Context, userId string) (*BalanceResponse, error)
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

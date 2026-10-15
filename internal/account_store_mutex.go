package internal

import (
	"context"
	"errors"
	"sync"

	"github.com/shopspring/decimal"
)

// ==========================================
// IMPLEMENTATION 2: Mutex-Protected Map Store
// ==========================================

type MutexAccountStore struct {
	mu       sync.RWMutex
	balances map[string]decimal.Decimal
}

func NewMutexAccountStore() *MutexAccountStore {
	return &MutexAccountStore{
		balances: make(map[string]decimal.Decimal),
	}
}

func (s *MutexAccountStore) Deposit(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("Amount needs to be > 0")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.balances[userId]
	newBalance := current.Add(amount)
	s.balances[userId] = newBalance

	return &BalanceResponse{UserId: userId, Balance: newBalance, Message: "Deposit successful"}, nil
}

func (s *MutexAccountStore) Withdraw(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("Amount needs to be > 0")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.balances[userId]
	if !exists {
		return nil, errors.New("Player not exist")
	}
	if current.LessThan(amount) {
		return nil, errors.New("Not enough Balance to withdraw")
	}

	newBalance := current.Sub(amount)
	s.balances[userId] = newBalance

	return &BalanceResponse{UserId: userId, Balance: newBalance, Message: "Withdraw successful"}, nil
}

func (s *MutexAccountStore) Balance(ctx context.Context, userId string) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	current, exists := s.balances[userId]
	if !exists {
		return nil, errors.New("Player not exist")
	}

	return &BalanceResponse{UserId: userId, Balance: current}, nil
}

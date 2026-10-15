package models

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
)

// ==========================================
// IMPLEMENTATION 1: Channel-Based Store
// ==========================================

type depositCommand struct {
	userId  string
	amount  decimal.Decimal
	replyTo chan storeResponse
}

type withdrawCommand struct {
	userId  string
	amount  decimal.Decimal
	replyTo chan storeResponse
}

type balanceCommand struct {
	userId  string
	replyTo chan storeResponse
}

type storeResponse struct {
	balance decimal.Decimal
	err     error
}

type ChannelAccountStore struct {
	depositChan  chan depositCommand
	withdrawChan chan withdrawCommand
	balanceChan  chan balanceCommand
}

func NewChannelAccountStore() *ChannelAccountStore {
	store := &ChannelAccountStore{
		depositChan:  make(chan depositCommand),
		withdrawChan: make(chan withdrawCommand),
		balanceChan:  make(chan balanceCommand),
	}
	go store.run()
	return store
}

func (s *ChannelAccountStore) run() {
	balances := make(map[string]decimal.Decimal)

	for {
		select {
		case cmd := <-s.depositChan:
			current := balances[cmd.userId]
			newBalance := current.Add(cmd.amount)
			balances[cmd.userId] = newBalance
			cmd.replyTo <- storeResponse{balance: newBalance}

		case cmd := <-s.withdrawChan:
			current, exists := balances[cmd.userId]
			if !exists {
				cmd.replyTo <- storeResponse{err: errors.New("Player not exist")}
				continue
			}
			if current.LessThan(cmd.amount) {
				cmd.replyTo <- storeResponse{err: errors.New("Not enough Balance to withdraw")}
				continue
			}
			newBalance := current.Sub(cmd.amount)
			balances[cmd.userId] = newBalance
			cmd.replyTo <- storeResponse{balance: newBalance}

		case cmd := <-s.balanceChan:
			current, exists := balances[cmd.userId]
			if !exists {
				cmd.replyTo <- storeResponse{err: errors.New("Player not exist")}
				continue
			}
			cmd.replyTo <- storeResponse{balance: current}
		}
	}
}

func (s *ChannelAccountStore) Deposit(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("Amount needs to be > 0")
	}

	replyChan := make(chan storeResponse)
	cmd := depositCommand{userId: userId, amount: amount, replyTo: replyChan}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s.depositChan <- cmd:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-replyChan:
		if res.err != nil {
			return nil, res.err
		}
		return &BalanceResponse{UserId: userId, Balance: res.balance, Message: "Deposit successful"}, nil
	}

}

func (s *ChannelAccountStore) Withdraw(ctx context.Context, userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("Amount needs to be > 0")
	}

	replyChan := make(chan storeResponse)
	cmd := withdrawCommand{userId: userId, amount: amount, replyTo: replyChan}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s.withdrawChan <- cmd:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-replyChan:
		if res.err != nil {
			return nil, res.err
		}
		return &BalanceResponse{UserId: userId, Balance: res.balance, Message: "Withdraw successful"}, nil
	}
}

func (s *ChannelAccountStore) Balance(ctx context.Context, userId string) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}

	replyChan := make(chan storeResponse)
	cmd := balanceCommand{userId: userId, replyTo: replyChan}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s.balanceChan <- cmd:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-replyChan:
		if res.err != nil {
			return nil, res.err
		}
		return &BalanceResponse{UserId: userId, Balance: res.balance}, nil
	}
}

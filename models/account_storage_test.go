package models

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestAccountStore_ConcurrentDeposits(t *testing.T) {
	store := NewChannelAccountStore()
	userId := "player_concurrent"

	const numGoroutines = 100
	depositAmount := decimal.NewFromFloat(10.00)

	// var wg sync.WaitGroup
	// wg.Add(numGoroutines)

	// Fire off 100 concurrent deposits of 10.00 each
	for i := 0; i < numGoroutines; i++ {
		go func() {
			// defer wg.Done()
			_, err := store.Deposit(userId, depositAmount)
			if err != nil {
				t.Errorf("unexpected error during concurrent deposit: %v", err)
			}
		}()
	}

	// wg.Wait()

	// Verify the final balance
	time.Sleep(5 * time.Second)
	res, err := store.Balance(userId)
	if err != nil {
		t.Fatalf("failed to get final balance: %v", err)
	}

	// Expected balance: 100 goroutines * 10.00 = 1000.00
	expectedBalance := decimal.NewFromFloat(1000.00)
	if !res.Balance.Equal(expectedBalance) {
		t.Errorf("expected final balance %s, got %s", expectedBalance.String(), res.Balance.String())
	}
}

func TestAccountStore_Deposit(t *testing.T) {
	store := NewChannelAccountStore()

	t.Run("successful deposit", func(t *testing.T) {
		res, err := store.Deposit("player1", decimal.NewFromFloat(100.50))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !res.Balance.Equal(decimal.NewFromFloat(100.50)) {
			t.Errorf("expected balance 100.50, got %s", res.Balance.String())
		}
	})

	t.Run("subsequent deposit adds to balance", func(t *testing.T) {
		res, err := store.Deposit("player1", decimal.NewFromFloat(50.00))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !res.Balance.Equal(decimal.NewFromFloat(150.50)) {
			t.Errorf("expected balance 150.50, got %s", res.Balance.String())
		}
	})

	t.Run("empty user id error", func(t *testing.T) {
		_, err := store.Deposit("", decimal.NewFromFloat(10.00))
		if err == nil {
			t.Error("expected error for empty user id, got nil")
		}
	})

	t.Run("non-positive amount error", func(t *testing.T) {
		_, err := store.Deposit("player1", decimal.Zero)
		if err == nil {
			t.Error("expected error for zero amount, got nil")
		}

		_, err = store.Deposit("player1", decimal.NewFromFloat(-10.00))
		if err == nil {
			t.Error("expected error for negative amount, got nil")
		}
	})
}

func TestAccountStore_Withdraw(t *testing.T) {
	store := NewChannelAccountStore()
	store.Deposit("player1", decimal.NewFromFloat(200.00))

	t.Run("successful withdrawal", func(t *testing.T) {
		res, err := store.Withdraw("player1", decimal.NewFromFloat(50.00))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := decimal.NewFromFloat(150.00)
		if !res.Balance.Equal(expected) {
			t.Errorf("expected balance %s, got %s", expected.String(), res.Balance.String())
		}
	})

	t.Run("insufficient funds error", func(t *testing.T) {
		_, err := store.Withdraw("player1", decimal.NewFromFloat(500.00))
		if err == nil {
			t.Error("expected error for insufficient funds, got nil")
		}
	})

	t.Run("non-existent player error", func(t *testing.T) {
		_, err := store.Withdraw("nonexistent", decimal.NewFromFloat(10.00))
		if err == nil {
			t.Error("expected error for non-existent player, got nil")
		}
	})

	t.Run("invalid withdrawal amount error", func(t *testing.T) {
		_, err := store.Withdraw("player1", decimal.NewFromFloat(-5.00))
		if err == nil {
			t.Error("expected error for negative withdrawal amount, got nil")
		}
	})
}

func TestAccountStore_Balance(t *testing.T) {
	store := NewChannelAccountStore()
	store.Deposit("player1", decimal.NewFromFloat(300.25))

	t.Run("retrieve existing balance", func(t *testing.T) {
		res, err := store.Balance("player1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := decimal.NewFromFloat(300.25)
		if !res.Balance.Equal(expected) {
			t.Errorf("expected balance %s, got %s", expected.String(), res.Balance.String())
		}
	})

	t.Run("retrieve non-existent balance error", func(t *testing.T) {
		_, err := store.Balance("unknown_user")
		if err == nil {
			t.Error("expected error for unknown user balance check, got nil")
		}
	})

	t.Run("empty user id error", func(t *testing.T) {
		_, err := store.Balance("")
		if err == nil {
			t.Error("expected error for empty user id, got nil")
		}
	})
}

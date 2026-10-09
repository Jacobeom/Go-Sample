package models

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/shopspring/decimal"
)

// AccountStore manages thread-safe player balances in memory.
type AccountStore struct {
	mu       sync.RWMutex
	balances map[string]decimal.Decimal
}

func NewAccountStore() *AccountStore {
	return &AccountStore{
		balances: make(map[string]decimal.Decimal),
	}
}

// TransactionRequest is the request payload for deposit and withdrawal operations.
type TransactionRequest struct {
	UserId string          `json:"user_id"`
	Amount decimal.Decimal `json:"amount"`
}

// BalanceResponse is the response payload for balance-related operations.
type BalanceResponse struct {
	UserId  string          `json:"player_id"`
	Balance decimal.Decimal `json:"balance"`
	Message string          `json:"message,omitempty"`
}

// ErrorResponse is the standard error response payload.
type ErrorResponse struct {
	Error string `json:"error"`
}

func (s *AccountStore) Deposit(userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("Amount neds to be > 0")
	}
	s.mu.Lock()
	currentBalance := s.balances[userId]
	newBalance := currentBalance.Add(amount)
	s.balances[userId] = newBalance
	s.mu.Unlock()

	return &BalanceResponse{
		UserId:  userId,
		Balance: newBalance,
		Message: "Deposit successful",
	}, nil
}

// HandleDeposit adds funds to a player's balance.
// @Summary      Deposit funds
// @Description  Add a specified decimal amount to a player's balance.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        request body TransactionRequest true "Deposit Request Payload"
// @Success      200 {object} BalanceResponse "Deposit successfully processed"
// @Failure      400 {object} ErrorResponse "Invalid JSON body, missing player_id, or non-positive amount"
// @Failure      405 {object} ErrorResponse "Method not allowed"
// @Router       /deposit [post]
func (s *AccountStore) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	res, errorDeposit := s.Deposit(req.UserId, req.Amount)
	if errorDeposit == nil {
		respondWithJSON(w, http.StatusOK, res)
	} else {
		respondWithError(w, http.StatusBadRequest, errorDeposit.Error())
	}
}

func (s *AccountStore) Withdraw(userId string, amount decimal.Decimal) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("Amount neds to be > 0")
	}

	s.mu.Lock()
	currentBalance, exists := s.balances[userId]
	if !exists {
		s.mu.Unlock()
		return nil, errors.New("Player not exist")
	}
	if currentBalance.LessThan(amount) {
		s.mu.Unlock()
		return nil, errors.New("Not enough Balance to withdraw")
	}
	newBalance := currentBalance.Sub(amount)
	s.balances[userId] = newBalance

	s.mu.Unlock()

	return &BalanceResponse{
		UserId:  userId,
		Balance: newBalance,
		Message: "Withdraw successful",
	}, nil
}

// HandleWithdraw deducts funds from a users's balance if sufficient funds exist.
// @Summary      Withdraw funds
// @Description  Deduct a specified decimal amount from a users's balance if sufficient balance is available.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        request body TransactionRequest true "Withdrawal Request Payload"
// @Success      200 {object} BalanceResponse "Withdrawal successfully processed"
// @Failure      400 {object} ErrorResponse "Insufficient funds, missing player_id, or invalid amount"
// @Failure      405 {object} ErrorResponse "Method not allowed"
// @Router       /withdraw [post]
func (s *AccountStore) HandleWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	res, errorWithdraw := s.Withdraw(req.UserId, req.Amount)

	if errorWithdraw != nil {
		respondWithError(w, http.StatusBadRequest, errorWithdraw.Error())
	}

	respondWithJSON(w, http.StatusOK, res)
}

// HandleGetBalance retrieves current balance for a user.
// @Summary      Get user balance
// @Description  Fetch current balance for a given user_id query parameter.
// @Tags         balance
// @Produce      json
// @Param        user_id query string true "user_id" example(player123)
// @Success      200 {object} BalanceResponse "User balance details"
// @Failure      400 {object} ErrorResponse "Missing player_id query parameter"
// @Failure      405 {object} ErrorResponse "Method not allowed"
// @Router       /balance [get]
func (s *AccountStore) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	userId := r.URL.Query().Get("user_id")
	res, errorWithdraw := s.Balance(userId)

	if errorWithdraw != nil {
		respondWithError(w, http.StatusBadRequest, errorWithdraw.Error())
	}

	respondWithJSON(w, http.StatusOK, res)
}

func (s *AccountStore) Balance(userId string) (*BalanceResponse, error) {
	if userId == "" {
		return nil, errors.New("user is required")
	}

	s.mu.Lock()
	currentBalance, exists := s.balances[userId]
	if !exists {
		s.mu.Unlock()
		return nil, errors.New("Player not exist")
	}

	s.mu.Unlock()

	return &BalanceResponse{
		UserId:  userId,
		Balance: currentBalance,
	}, nil
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}

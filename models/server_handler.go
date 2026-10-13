package models

import (
	"encoding/json"
	"net/http"
)

// ==========================================
// HTTP SERVER & HANDLERS (Polymorphic)
// ==========================================

type Server struct {
	store AccountStore
}

func NewServer(store AccountStore) *Server {
	return &Server{store: store}
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
func (s *Server) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	res, err := s.store.Deposit(req.UserId, req.Amount)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, res)
}

// HandleWithdraw deducts funds from a user's balance if sufficient funds exist.
// @Summary      Withdraw funds
// @Description  Deduct a specified decimal amount from a user's balance if sufficient balance is available.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        request body TransactionRequest true "Withdrawal Request Payload"
// @Success      200 {object} BalanceResponse "Withdrawal successfully processed"
// @Failure      400 {object} ErrorResponse "Insufficient funds, missing player_id, or invalid amount"
// @Failure      405 {object} ErrorResponse "Method not allowed"
// @Router       /withdraw [post]
func (s *Server) HandleWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	res, err := s.store.Withdraw(req.UserId, req.Amount)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
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
func (s *Server) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	userId := r.URL.Query().Get("user_id")
	res, err := s.store.Balance(userId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, res)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}

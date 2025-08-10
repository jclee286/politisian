package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionStore는 이제 세션 토큰과 '지갑 주소'를 매핑합니다.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string // key: sessionToken, value: walletAddress
}

func (s *SessionStore) Set(token, address string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = address
}

func (s *SessionStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	address, exists := s.sessions[token]
	return address, exists
}

var sessionStore = &SessionStore{
	sessions: make(map[string]string),
}

type contextKey string

const userWalletAddressKey contextKey = "address"

// handleWalletLogin은 지갑 주소를 받아 로그인을 처리하고 세션 쿠키를 발급합니다.
func handleWalletLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userAddress := req.Address
	if userAddress == "" {
		http.Error(w, "Wallet address is required", http.StatusBadRequest)
		return
	}

	// 로그인 요청이 오면, 항상 새로운 세션 토큰을 발급합니다.
	sessionToken := uuid.New().String()
	sessionStore.Set(sessionToken, userAddress)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
} 
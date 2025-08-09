package server

import (
	"context"
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

// authMiddleware는 이제 쿠키에서 지갑 주소를 확인합니다.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			// 쿠키가 없으면 로그인 페이지로 리디렉션하거나 401 오류를 반환할 수 있습니다.
			// 여기서는 API 요청이므로 401을 반환합니다.
			http.Error(w, "Unauthorized: No session token", http.StatusUnauthorized)
			return
		}

		sessionToken := sessionCookie.Value
		address, exists := sessionStore.Get(sessionToken)
		if !exists {
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		}

		// 컨텍스트에 지갑 주소를 추가합니다.
		ctx := context.WithValue(r.Context(), userWalletAddressKey, address)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
} 
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

// handleSocialLogin은 Web3Auth 소셜 로그인 데이터를 받아 로그인을 처리하고 세션 쿠키를 발급합니다.
func handleSocialLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Provider     string `json:"provider"`
		UserId       string `json:"userId"`
		ProfileImage string `json:"profileImage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 이메일은 필수
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// 소셜 로그인의 경우 이메일을 사용자 ID로 사용
	userID := req.Email

	// 새로운 세션 토큰 발급
	sessionToken := uuid.New().String()
	sessionStore.Set(sessionToken, userID)

	// 세션 쿠키 설정
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// 성공 응답과 함께 사용자 정보 반환
	response := map[string]interface{}{
		"status": "success",
		"user": map[string]string{
			"name":         req.Name,
			"email":        req.Email,
			"provider":     req.Provider,
			"userId":       req.UserId,
			"profileImage": req.ProfileImage,
		},
		"sessionToken": sessionToken,
		"isNewUser":    true, // TODO: 실제로는 DB에서 사용자 존재 여부 확인
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
} 
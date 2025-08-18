package server

import (
	"encoding/json"
	"net/http"
)

// handleWalletLogin은 지갑 로그인을 처리합니다 (레거시 - 사용하지 않음).
func handleWalletLogin(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "이 로그인 방법은 더 이상 지원되지 않습니다. 새로운 로그인을 사용해주세요.", http.StatusGone)
}

// handleSocialLogin은 소셜 로그인을 처리합니다 (레거시 - 사용하지 않음).
func handleSocialLogin(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "소셜 로그인은 더 이상 지원되지 않습니다. 새로운 회원가입/로그인을 사용해주세요.", http.StatusGone)
}

// handleSessionInfo는 세션 정보를 반환합니다 (레거시 지원).
func handleSessionInfo(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "세션을 찾을 수 없습니다", http.StatusUnauthorized)
		return
	}

	sessionData, exists := sessionStore.GetSessionData(sessionCookie.Value)
	if !exists {
		http.Error(w, "세션 데이터를 찾을 수 없습니다", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"userId":        sessionData.UserID,
		"email":         sessionData.Email,
		"name":          sessionData.Name,
		"walletAddress": sessionData.WalletAddress,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
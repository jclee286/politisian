package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	ptypes "politisian/pkg/types"
)

// SessionData는 세션에 저장할 사용자 정보를 정의합니다.
type SessionData struct {
	UserID        string
	Email         string
	WalletAddress string
	Name          string
	ProfileImage  string
}

// SessionStore는 세션 토큰과 사용자 정보를 매핑합니다.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
}

func (s *SessionStore) Set(token string, data *SessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = data
}

func (s *SessionStore) SetUserID(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if data, exists := s.sessions[token]; exists {
		data.UserID = userID
	} else {
		s.sessions[token] = &SessionData{UserID: userID}
	}
}

func (s *SessionStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if data, exists := s.sessions[token]; exists {
		return data.UserID, true
	}
	return "", false
}

func (s *SessionStore) GetSessionData(token string) (*SessionData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, exists := s.sessions[token]
	return data, exists
}

var sessionStore = &SessionStore{
	sessions: make(map[string]*SessionData),
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
	sessionStore.SetUserID(sessionToken, userAddress)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(1 * time.Hour),  // 1시간으로 변경
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleSocialLogin은 구글 OAuth 로그인 데이터를 받아 로그인을 처리하고 세션 쿠키를 발급합니다.
func handleSocialLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		Provider      string `json:"provider"`
		UserId        string `json:"userId"`
		ProfileImage  string `json:"profileImage"`
		PIN           string `json:"pin"`  // 6자리 PIN 번호
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

	// PIN을 이용한 지갑 주소 생성 및 블록체인 계정 생성
	walletAddress, err := generateWalletFromPin(req.Email, req.PIN)
	if err != nil {
		http.Error(w, fmt.Sprintf("지갑 생성 실패: %v", err), http.StatusInternalServerError)
		return
	}

	// 새로운 세션 토큰 발급
	sessionToken := uuid.New().String()
	log.Printf("소셜 로그인: 사용자 %s를 위한 세션 토큰 생성", userID)
	
	sessionData := &SessionData{
		UserID:        userID,
		Email:         req.Email,
		WalletAddress: walletAddress,
		Name:          req.Name,
		ProfileImage:  req.ProfileImage,
	}
	sessionStore.Set(sessionToken, sessionData)
	
	log.Printf("세션 저장 완료 - 사용자: %s", userID)

	// 블록체인에 계정 생성 (존재하지 않는 경우에만)
	// 계정 생성 결과로 신규/기존 회원 여부를 판별
	isNewUser := true
	if err := createBlockchainAccount(userID, req.Email, walletAddress); err != nil {
		// 이미 존재하는 계정인 경우 기존 회원으로 판단
		if err.Error() == "account already exists" {
			isNewUser = false
			log.Printf("기존 회원 로그인: %s", userID)
		} else {
			// 실제 에러인 경우
			json.NewEncoder(w).Encode(map[string]string{
				"status": "login_success_but_account_creation_failed",
				"error":  err.Error(),
			})
			return
		}
	} else {
		log.Printf("신규 회원 가입: %s", userID)
	}

	// 세션 쿠키 설정
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(1 * time.Hour),  // 1시간으로 변경
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	log.Printf("세션 쿠키 설정 완료 - 사용자: %s", userID)

	// 성공 응답과 함께 사용자 정보 반환
	response := map[string]interface{}{
		"status": "success",
		"user": map[string]string{
			"name":          req.Name,
			"email":         req.Email,
			"provider":      req.Provider,
			"userId":        req.UserId,
			"profileImage":  req.ProfileImage,
			"walletAddress": walletAddress,
		},
		"sessionToken": sessionToken,
		"isNewUser":    isNewUser,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleSessionInfo는 현재 세션의 사용자 정보와 지갑 주소를 반환합니다.
func handleSessionInfo(w http.ResponseWriter, r *http.Request) {
	// 세션 토큰 확인
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "세션 토큰이 없습니다", http.StatusUnauthorized)
		return
	}

	// 세션 스토어에서 사용자 정보 가져오기
	sessionData, exists := sessionStore.GetSessionData(cookie.Value)
	if !exists {
		http.Error(w, "세션 데이터를 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"userId":        sessionData.UserID,
		"email":         sessionData.Email,
		"walletAddress": sessionData.WalletAddress,
		"name":          sessionData.Name,
		"profileImage":  sessionData.ProfileImage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGenerateWallet은 현재 세션 사용자를 위한 지갑을 생성합니다.
func handleGenerateWallet(w http.ResponseWriter, r *http.Request) {
	// 세션 토큰 확인
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "세션 토큰이 없습니다", http.StatusUnauthorized)
		return
	}

	// 세션에서 사용자 정보 가져오기
	sessionData, exists := sessionStore.GetSessionData(cookie.Value)
	if !exists {
		http.Error(w, "세션 데이터를 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	walletAddress := sessionData.WalletAddress

	response := map[string]interface{}{
		"walletAddress": walletAddress,
		"status": "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createBlockchainAccount은 블록체인에 새로운 사용자 계정을 생성합니다.
func createBlockchainAccount(userID, email, walletAddress string) error {
	// 먼저 계정이 이미 존재하는지 확인
	queryPath := fmt.Sprintf("/account?address=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		log.Printf("Error checking existing account: %v", err)
		// 쿼리 에러는 무시하고 계정 생성 시도
	} else if res.Response.Code == 0 {
		// 계정이 이미 존재함
		log.Printf("Account already exists for user: %s", userID)
		return fmt.Errorf("account already exists")
	}

	// 계정 생성 트랜잭션 생성
	txData := ptypes.TxData{
		Action:        "create_profile",
		UserID:        userID,
		Email:         email,
		WalletAddress: walletAddress,
	}
	txBytes, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("failed to marshal tx data: %v", err)
	}

	// 트랜잭션 전송
	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		return fmt.Errorf("failed to create blockchain account: %v", err)
	}

	log.Printf("Successfully created blockchain account for user: %s", userID)
	return nil
}

// generateWalletFromPin은 이메일과 PIN을 조합하여 지갑 주소를 생성합니다.
func generateWalletFromPin(email, pin string) (string, error) {
	// PIN 검증 (6자리 숫자)
	if len(pin) != 6 {
		return "", fmt.Errorf("PIN must be 6 digits")
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("PIN must contain only numbers")
		}
	}

	// 환경변수에서 솔트 가져오기
	baseSalt := os.Getenv("WALLET_SALT")
	if baseSalt == "" {
		baseSalt = "default_salt_2025" // 기본값 (프로덕션에서는 반드시 환경변수 설정)
	}

	// 앱 고유 식별자
	appSalt := "정치인공화국_wallet"

	// 솔트 + 이메일 + PIN 조합으로 지갑 주소 생성
	combined := email + baseSalt + pin + appSalt
	hash := sha256.Sum256([]byte(combined))
	walletAddress := hex.EncodeToString(hash[:])

	log.Printf("Generated wallet address for user %s: %s", email, walletAddress[:20]+"...")
	return walletAddress, nil
}
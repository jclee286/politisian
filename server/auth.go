package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	ptypes "github.com/jclee286/politisian/pkg/types"

	"golang.org/x/crypto/bcrypt"
)

// SessionData는 세션에 저장되는 사용자 정보입니다.
type SessionData struct {
	UserID        string    `json:"user_id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	WalletAddress string    `json:"wallet_address"`
	CreatedAt     time.Time `json:"created_at"`
}

// SessionStore는 세션 관리를 위한 구조체입니다.
type SessionStore struct {
	sessions     map[string]string      // token -> userID
	sessionData  map[string]SessionData // token -> SessionData
	mu           sync.RWMutex
}

// NewSessionStore는 새로운 세션 스토어를 생성합니다.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions:    make(map[string]string),
		sessionData: make(map[string]SessionData),
	}
}

// Set은 세션 토큰과 사용자 ID를 저장합니다.
func (s *SessionStore) Set(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = userID
}

// Get은 세션 토큰으로 사용자 ID를 조회합니다.
func (s *SessionStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, exists := s.sessions[token]
	return userID, exists
}

// SetSessionData는 세션 데이터를 저장합니다.
func (s *SessionStore) SetSessionData(token string, data SessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionData[token] = data
}

// GetSessionData는 세션 데이터를 조회합니다.
func (s *SessionStore) GetSessionData(token string) (SessionData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, exists := s.sessionData[token]
	return data, exists
}

// 전역 세션 스토어
var sessionStore = NewSessionStore()

// handleSignup는 전통적 회원가입을 처리합니다.
func handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ptypes.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "잘못된 요청 형식", http.StatusBadRequest)
		return
	}

	// 입력 검증
	if req.Email == "" || req.Password == "" || req.Nickname == "" || req.PIN == "" {
		http.Error(w, "모든 필드를 입력해주세요", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "비밀번호는 8자 이상이어야 합니다", http.StatusBadRequest)
		return
	}

	if len(req.PIN) != 6 {
		http.Error(w, "PIN은 6자리여야 합니다", http.StatusBadRequest)
		return
	}

	if len(req.Politicians) != 3 {
		http.Error(w, "정확히 3명의 정치인을 선택해야 합니다", http.StatusBadRequest)
		return
	}

	// 이메일 중복 확인
	if userExists(req.Email) {
		http.Error(w, "이미 등록된 이메일입니다", http.StatusConflict)
		return
	}

	// 비밀번호 해시
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "서버 오류", http.StatusInternalServerError)
		return
	}

	// PIN 해시 (SHA256)
	pinHash := sha256.Sum256([]byte(req.PIN))
	pinHashStr := hex.EncodeToString(pinHash[:])

	// 사용자 ID 생성
	userID := generateUserID()

	// 사용자 정보 저장
	user := &ptypes.User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		Nickname:     req.Nickname,
		PIN:          pinHashStr,
		CreatedAt:    time.Now().Unix(),
		IsActive:     true,
	}

	// 사용자 저장 (메모리 또는 파일에 저장, 나중에 블록체인으로 이동 가능)
	if err := saveUser(user); err != nil {
		http.Error(w, "사용자 저장 실패", http.StatusInternalServerError)
		return
	}

	// 지갑 주소 생성 (PIN 기반)
	walletAddress := generateWalletAddress(req.PIN)

	// 블록체인에 계정 생성
	if err := createBlockchainAccount(userID, req.Email, walletAddress, req.Politicians); err != nil {
		http.Error(w, "블록체인 계정 생성 실패", http.StatusInternalServerError)
		return
	}

	// 초기 코인 지급을 위한 update_supporters 트랜잭션 추가 전송
	if err := sendInitialCoinsTransaction(userID, req.Politicians); err != nil {
		log.Printf("Warning: 초기 코인 지급 실패하지만 회원가입은 완료: %v", err)
		// 초기 코인 지급이 실패해도 회원가입은 성공으로 처리
	}

	// 성공 응답
	response := map[string]interface{}{
		"success": true,
		"message": "회원가입이 완료되었습니다",
		"user_id": userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLogin은 전통적 로그인을 처리합니다.
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ptypes.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "잘못된 요청 형식", http.StatusBadRequest)
		return
	}

	// 입력 검증
	if req.Email == "" || req.Password == "" {
		http.Error(w, "이메일과 비밀번호를 입력해주세요", http.StatusBadRequest)
		return
	}

	// 사용자 조회
	user, exists := getUser(req.Email)
	if !exists {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// 비밀번호 검증
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 세션 생성
	sessionToken := generateSessionToken()
	
	// 세션 데이터 저장
	sessionData := SessionData{
		UserID:        user.ID,
		Email:         user.Email,
		Name:          user.Nickname,
		WalletAddress: generateWalletAddress(user.PIN), // PIN에서 지갑 주소 재생성
		CreatedAt:     time.Now(),
	}
	
	sessionStore.Set(sessionToken, user.ID)
	sessionStore.SetSessionData(sessionToken, sessionData)

	// 쿠키 설정
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   86400 * 7, // 7일
		HttpOnly: true,
		Secure:   false, // 개발 환경에서는 false
		SameSite: http.SameSiteLaxMode,
	})

	// 성공 응답
	response := ptypes.LoginResponse{
		Success: true,
		Message: "로그인 성공",
		UserID:  user.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 헬퍼 함수들

// generateUserID는 고유한 사용자 ID를 생성합니다.
func generateUserID() string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return fmt.Sprintf("user_%d_%x", time.Now().UnixNano(), randBytes[:8])
}

// generateSessionToken은 세션 토큰을 생성합니다.
func generateSessionToken() string {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	return hex.EncodeToString(randBytes)
}

// generateWalletAddress는 PIN을 기반으로 지갑 주소를 생성합니다.
func generateWalletAddress(pin string) string {
	hash := sha256.Sum256([]byte("wallet_" + pin))
	return hex.EncodeToString(hash[:])
}

// 사용자 저장/조회 함수들 (간단한 메모리 저장소, 나중에 개선 가능)
var userStore = make(map[string]*ptypes.User) // email -> user

func saveUser(user *ptypes.User) error {
	userStore[user.Email] = user
	return nil
}

func getUser(email string) (*ptypes.User, bool) {
	user, exists := userStore[email]
	return user, exists
}

func userExists(email string) bool {
	_, exists := userStore[email]
	return exists
}

// createBlockchainAccount는 블록체인에 계정을 생성합니다.
func createBlockchainAccount(userID, email, walletAddress string, politicians []string) error {
	// 고유한 트랜잭션 ID 생성
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	txID := fmt.Sprintf("%s-signup-%d-%x", userID, time.Now().UnixNano(), randBytes)

	txData := ptypes.TxData{
		TxID:          txID,
		Action:        "create_profile",
		UserID:        userID,
		Email:         email,
		WalletAddress: walletAddress,
		Politicians:   politicians,
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("transaction marshal error: %v", err)
	}

	return broadcastAndCheckTx(context.Background(), txBytes)
}

// sendInitialCoinsTransaction은 초기 코인 지급을 위한 update_supporters 트랜잭션을 전송합니다.
func sendInitialCoinsTransaction(userID string, politicians []string) error {
	// 고유한 트랜잭션 ID 생성
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	txID := fmt.Sprintf("%s-initial-coins-%d-%x", userID, time.Now().UnixNano(), randBytes)

	txData := ptypes.TxData{
		TxID:        txID,
		Action:      "update_supporters",
		UserID:      userID,
		Politicians: politicians,
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("initial coins transaction marshal error: %v", err)
	}

	return broadcastAndCheckTx(context.Background(), txBytes)
}

// handleVerifyPIN은 사용자의 PIN을 검증합니다.
func handleVerifyPIN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 세션에서 사용자 정보 가져오기
	sessionCookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "세션이 없습니다", http.StatusUnauthorized)
		return
	}

	_, exists := sessionStore.Get(sessionCookie.Value)
	if !exists {
		http.Error(w, "유효하지 않은 세션입니다", http.StatusUnauthorized)
		return
	}

	var req struct {
		PIN string `json:"pin"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "잘못된 요청 형식", http.StatusBadRequest)
		return
	}

	// 사용자 조회 (userID로 이메일을 찾아서 사용자 정보 조회)
	// 현재는 간단히 세션에서 이메일을 가져옴
	sessionData, exists := sessionStore.GetSessionData(sessionCookie.Value)
	if !exists {
		http.Error(w, "세션 데이터가 없습니다", http.StatusUnauthorized)
		return
	}

	user, exists := getUser(sessionData.Email)
	if !exists {
		http.Error(w, "사용자를 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	// PIN 검증
	pinHash := sha256.Sum256([]byte(req.PIN))
	pinHashStr := hex.EncodeToString(pinHash[:])

	if pinHashStr != user.PIN {
		http.Error(w, "잘못된 PIN입니다", http.StatusUnauthorized)
		return
	}

	// 성공 응답
	response := map[string]interface{}{
		"success": true,
		"message": "PIN 검증 성공",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
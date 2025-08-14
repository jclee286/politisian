package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	ptypes "politisian/pkg/types"

	"github.com/cometbft/cometbft/abci/types"
)

// broadcastAndCheckTx, handleUserProfile, handleGetPolitisians는 이전과 거의 동일하게 유지

func broadcastAndCheckTx(ctx context.Context, txBytes []byte) error {
	res, err := blockchainClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		log.Printf("Error broadcasting tx: %v", err)
		return fmt.Errorf("RPC 오류: %v", err)
	}
	if res.Code != types.CodeTypeOK {
		log.Printf("Tx failed. Code: %d, Log: %s", res.Code, res.Log)
		return fmt.Errorf("트랜잭션 실패: %s (코드: %d)", res.Log, res.Code)
	}
	log.Printf("Tx broadcast successful. Hash: %s", res.Hash.String())
	return nil
}

func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/user/profile request")
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		http.Error(w, "사용자 ID를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	// ABCI 쿼리를 통해 사용자 계정 정보 가져오기
	queryPath := fmt.Sprintf("/account?address=%s", userID)
	log.Printf("Querying ABCI for user profile: %s", queryPath)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		log.Printf("Error querying ABCI for user profile: %v", err)
		// 블록체인에서 조회 실패 시 세션 데이터로 대체 시도
		handleUserProfileFromSession(w, r, userID)
		return
	}
	if res.Response.Code != 0 {
		log.Printf("Account not found in blockchain for user %s, trying session data", userID)
		// 블록체인에 계정이 없으면 세션 데이터로 대체 시도
		handleUserProfileFromSession(w, r, userID)
		return
	}

	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		log.Printf("Error unmarshalling user profile: %v", err)
		// 파싱 실패 시 세션 데이터로 대체 시도
		handleUserProfileFromSession(w, r, userID)
		return
	}
	
	log.Printf("Successfully fetched and sending profile for user %s", userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// 세션 데이터로 프로필 정보를 반환하는 함수
func handleUserProfileFromSession(w http.ResponseWriter, r *http.Request, userID string) {
	log.Printf("Attempting to get profile from session for user %s", userID)
	
	// 쿠키에서 세션 ID 가져오기
	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Printf("No session cookie found for user %s", userID)
		http.Error(w, "세션을 찾을 수 없습니다", http.StatusUnauthorized)
		return
	}

	// 세션 데이터 가져오기
	sessionData, exists := sessionStore.GetSessionData(cookie.Value)
	if !exists {
		log.Printf("No session data found for user %s", userID)
		http.Error(w, "세션 데이터를 찾을 수 없습니다", http.StatusUnauthorized)
		return
	}

	// 세션 데이터를 Account 형태로 변환
	account := ptypes.Account{
		Address:     userID,
		Email:       sessionData.Email,
		Wallet:      sessionData.WalletAddress,
		Politicians: []string{}, // 세션에는 정치인 정보가 없으므로 빈 배열
	}

	log.Printf("Successfully returning session-based profile for user %s", userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func handleGetPolitisians(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/politisian/list request")
	res, err := blockchainClient.ABCIQuery(context.Background(), "/politisian/list", nil)
	if err != nil {
		log.Printf("Error querying for politisian list: %v", err)
		http.Error(w, fmt.Sprintf("블록체인 쿼리 실패: %v", err), http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		log.Printf("Failed to get politisian list from app. Code: %d, Log: %s", res.Response.Code, res.Response.Log)
		http.Error(w, "정치인 목록 조회에 실패했습니다.", http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched politisian list.")
	w.Header().Set("Content-Type", "application/json")
	w.Write(res.Response.Value)
}

// handleProfileSave는 사용자의 프로필을 저장하는 요청을 처리합니다.
func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/profile/save request")
	userID, _ := r.Context().Value("userID").(string)
	email, _ := r.Context().Value("email").(string)
	walletAddress, _ := r.Context().Value("walletAddress").(string)
	
	var reqBody ptypes.ProfileSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Printf("Error decoding profile save request: %v", err)
		http.Error(w, "잘못된 요청", http.StatusBadRequest)
		return
	}
	log.Printf("User %s is saving profile - nickname: %s, politicians: %v", userID, reqBody.Nickname, reqBody.Politisians)

	// 먼저 기존 계정이 있는지 확인
	queryPath := fmt.Sprintf("/account?address=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	
	var action string
	if err != nil || res.Response.Code != 0 {
		// 계정이 없으면 새로 생성
		action = "create_profile"
		log.Printf("Creating new profile for user %s", userID)
	} else {
		// 계정이 있으면 업데이트
		action = "update_supporters"
		log.Printf("Updating existing profile for user %s", userID)
	}
	
	// 고유한 트랜잭션 ID 생성 (타임스탬프 + 사용자ID + 랜덤요소)
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	txID := fmt.Sprintf("%s-%d-%x", userID, time.Now().UnixNano(), randBytes)

	txData := ptypes.TxData{
		TxID:          txID,
		Action:        action,
		UserID:        userID,
		Email:         email,
		WalletAddress: walletAddress,
		Politicians:   reqBody.Politisians,
	}
	txBytes, _ := json.Marshal(txData)

	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		log.Printf("Error broadcasting profile save transaction: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Profile save successful for user %s", userID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("프로필이 성공적으로 저장되었습니다"))
}


// handleProposePolitisian는 새로운 정치인을 등록 제안하는 요청을 처리합니다.
func handleProposePolitician(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/politisian/propose request")
	userID, _ := r.Context().Value("userID").(string)
	var reqBody struct {
		Name   string `json:"name"`
		Region string `json:"region"`
		Party  string `json:"party"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청", http.StatusBadRequest)
		return
	}
	log.Printf("User %s is proposing a new politisian: %s", userID, reqBody.Name)

	txData := ptypes.TxData{
		Action:         "propose_politician",
		UserID:         userID,
		PoliticianName: reqBody.Name,
		Region:         reqBody.Region,
		Party:          reqBody.Party,
	}
	txBytes, _ := json.Marshal(txData)

	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

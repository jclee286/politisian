package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
		log.Printf("Account not found in blockchain for user %s, creating basic account", userID)
		// 기존 회원인 경우 기본 계정 생성
		createBasicAccount(userID, r)
		
		// 다시 조회 시도
		res, err = blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
		if err != nil || res.Response.Code != 0 {
			log.Printf("Still failed to create/find account, falling back to session data")
			handleUserProfileFromSession(w, r, userID)
			return
		}
		// 성공하면 계속 진행
	}

	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		log.Printf("Error unmarshalling user profile: %v", err)
		// 파싱 실패 시 세션 데이터로 대체 시도
		handleUserProfileFromSession(w, r, userID)
		return
	}
	
	// Account를 ProfileInfoResponse로 변환
	totalCoins := int64(0)
	for _, coins := range account.PoliticianCoins {
		totalCoins += coins
	}
	
	response := ptypes.ProfileInfoResponse{
		Email:           account.Email,
		Wallet:          account.Wallet,
		Politisians:     account.Politicians,
		Balance:         totalCoins,                // 총 코인 잔액
		ReferralCredits: account.ReferralCredits,
		PoliticianCoins: account.PoliticianCoins,   // 정치인별 코인 보유량
		TotalCoins:      totalCoins,                // 총 코인 수 (편의용)
	}
	
	log.Printf("Successfully fetched and sending profile for user %s (total coins: %d)", userID, totalCoins)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 세션 데이터로 프로필 정보를 반환하는 함수
func handleUserProfileFromSession(w http.ResponseWriter, r *http.Request, userID string) {
	log.Printf("Attempting to get profile from session for user %s", userID)
	
	// 쿠키에서 세션 토큰 가져오기 (session_token으로 통일)
	cookie, err := r.Cookie("session_token")
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

	// 세션 데이터를 ProfileInfoResponse 형태로 변환
	response := ptypes.ProfileInfoResponse{
		Email:           sessionData.Email,
		Wallet:          sessionData.WalletAddress,
		Politisians:     []string{},                    // 세션에는 정치인 정보가 없으므로 빈 배열
		Balance:         0,                             // 세션에는 코인 정보가 없음
		ReferralCredits: 0,                             // 세션에는 크레딧 정보가 없음
		PoliticianCoins: make(map[string]int64),        // 빈 맵
		TotalCoins:      0,                             // 0개
	}

	log.Printf("Successfully returning session-based profile for user %s", userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetPolitisians(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/politisian/list request")
	res, err := blockchainClient.ABCIQuery(context.Background(), "/proposals/list", nil)
	if err != nil {
		log.Printf("Error querying for proposals list: %v", err)
		http.Error(w, fmt.Sprintf("블록체인 쿼리 실패: %v", err), http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		log.Printf("Failed to get proposals list from app. Code: %d, Log: %s", res.Response.Code, res.Response.Log)
		http.Error(w, "제안 목록 조회에 실패했습니다.", http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched proposals list.")
	w.Header().Set("Content-Type", "application/json")
	w.Write(res.Response.Value)
}

// handleGetRegisteredPoliticians는 등록된 정치인 목록을 조회합니다.
func handleGetRegisteredPoliticians(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/politisian/registered request")
	res, err := blockchainClient.ABCIQuery(context.Background(), "/politisian/list", nil)
	if err != nil {
		log.Printf("Error querying for politicians list: %v", err)
		http.Error(w, fmt.Sprintf("블록체인 쿼리 실패: %v", err), http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		log.Printf("Failed to get politicians list from app. Code: %d, Log: %s", res.Response.Code, res.Response.Log)
		http.Error(w, "등록된 정치인 목록 조회에 실패했습니다.", http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched registered politicians list.")
	w.Header().Set("Content-Type", "application/json")
	w.Write(res.Response.Value)
}

// handleVoteOnProposal는 제안에 대한 투표를 처리합니다.
func handleVoteOnProposal(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle vote on proposal request")
	userID, _ := r.Context().Value("userID").(string)
	
	// URL에서 proposal ID 추출 (예: /api/proposals/123/vote)
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "잘못된 요청 경로", http.StatusBadRequest)
		return
	}
	proposalID := parts[3] // proposals/{id}/vote에서 {id} 부분
	
	var reqBody struct {
		Vote bool `json:"vote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청", http.StatusBadRequest)
		return
	}
	
	log.Printf("User %s is voting %v on proposal %s", userID, reqBody.Vote, proposalID)

	// 고유한 트랜잭션 ID 생성
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	txID := fmt.Sprintf("%s-vote-%d-%x", userID, time.Now().UnixNano(), randBytes)

	txData := ptypes.TxData{
		TxID:       txID,
		Action:     "vote_on_proposal",
		UserID:     userID,
		ProposalID: proposalID,
		Vote:       reqBody.Vote,
	}
	txBytes, _ := json.Marshal(txData)

	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		log.Printf("Error broadcasting vote transaction: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Vote successful for user %s on proposal %s", userID, proposalID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("투표가 성공적으로 처리되었습니다"))
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
		Referrer:      reqBody.Referrer,
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
		Name     string `json:"name"`
		Region   string `json:"region"`
		Party    string `json:"party"`
		IntroUrl string `json:"introUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청", http.StatusBadRequest)
		return
	}
	log.Printf("User %s is proposing a new politisian: %s", userID, reqBody.Name)

	// 고유한 트랜잭션 ID 생성 (타임스탬프 + 사용자ID + 랜덤요소)
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	txID := fmt.Sprintf("%s-propose-%d-%x", userID, time.Now().UnixNano(), randBytes)

	txData := ptypes.TxData{
		TxID:           txID,
		Action:         "propose_politician",
		UserID:         userID,
		PoliticianName: reqBody.Name,
		Region:         reqBody.Region,
		Party:          reqBody.Party,
		IntroUrl:       reqBody.IntroUrl,
	}
	txBytes, _ := json.Marshal(txData)

	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// createBasicAccount는 기존 회원을 위한 기본 계정을 생성합니다.
func createBasicAccount(userID string, r *http.Request) error {
	log.Printf("Creating basic account for existing user %s", userID)
	
	// 세션에서 이메일과 지갑 주소 가져오기
	email := r.Context().Value("email")
	walletAddress := r.Context().Value("walletAddress")
	
	var emailStr, walletStr string
	if email != nil {
		emailStr = email.(string)
	}
	if walletAddress != nil {
		walletStr = walletAddress.(string)
	}
	
	// 고유한 트랜잭션 ID 생성
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	txID := fmt.Sprintf("%s-basic-%d-%x", userID, time.Now().UnixNano(), randBytes)

	txData := ptypes.TxData{
		TxID:          txID,
		Action:        "create_profile",
		UserID:        userID,
		Email:         emailStr,
		WalletAddress: walletStr,
		Politicians:   []string{}, // 빈 정치인 목록으로 시작
	}
	
	txBytes, err := json.Marshal(txData)
	if err != nil {
		log.Printf("Error marshaling basic account transaction: %v", err)
		return err
	}

	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		log.Printf("Error broadcasting basic account transaction: %v", err)
		return err
	}
	
	log.Printf("Basic account created successfully for user %s", userID)
	return nil
}

// handleClaimReward는 추천 크레딧을 사용하는 핸들러입니다.
func handleClaimReward(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/rewards/claim request")
	userID, _ := r.Context().Value("userID").(string)
	
	// 현재 사용자 계정 정보 조회
	queryPath := fmt.Sprintf("/account?address=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		log.Printf("Error querying account for reward claim: %v", err)
		http.Error(w, "계정 정보를 조회할 수 없습니다", http.StatusInternalServerError)
		return
	}
	
	if res.Response.Code != 0 {
		log.Printf("Account not found for reward claim: %s", userID)
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}
	
	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		log.Printf("Error unmarshaling account data: %v", err)
		http.Error(w, "계정 데이터 파싱 오류", http.StatusInternalServerError)
		return
	}
	
	// 사용 가능한 크레딧이 있는지 확인
	if account.ReferralCredits <= 0 {
		log.Printf("No referral credits available for user %s", userID)
		http.Error(w, "사용 가능한 추천 크레딧이 없습니다", http.StatusBadRequest)
		return
	}
	
	// 크레딧 사용 트랜잭션 생성
	txData := ptypes.TxData{
		Action: "claim_referral_reward",
		UserID: userID,
	}
	txBytes, _ := json.Marshal(txData)
	
	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		log.Printf("Error broadcasting claim reward transaction: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("추천 크레딧이 성공적으로 사용되었습니다"))
}

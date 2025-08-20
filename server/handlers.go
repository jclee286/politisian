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

	ptypes "github.com/jclee286/politisian/pkg/types"

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
		USDTBalance:   account.USDTBalance,     // 테더코인 잔액
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
		USDTBalance:   0,                             // 테더코인 잔액 (기본값 0)
	}

	log.Printf("Successfully returning session-based profile for user %s", userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetPolitisians(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/github.com/jclee286/politisian/list request")
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
	log.Println("Attempting to handle /api/github.com/jclee286/politisian/registered request")
	res, err := blockchainClient.ABCIQuery(context.Background(), "/github.com/jclee286/politisian/list", nil)
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
	log.Println("Attempting to handle /api/github.com/jclee286/politisian/propose request")
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
	
	// 기존 사용자 정보에서 정치인 목록 가져오기
	var selectedPoliticians []string
	userQueryPath := fmt.Sprintf("/user?id=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), userQueryPath, nil)
	if err == nil && res.Response.Code == 0 {
		var user ptypes.User
		if err := json.Unmarshal(res.Response.Value, &user); err == nil {
			log.Printf("Found existing user data for %s", userID)
			// User 구조체에는 정치인 정보가 없으므로 기본 정치인들로 설정
			selectedPoliticians = []string{"이재명", "윤석열", "이낙연"} // 기본 정치인들
		}
	} else {
		log.Printf("No existing user data found for %s, using default politicians", userID)
		selectedPoliticians = []string{"이재명", "윤석열", "이낙연"} // 기본 정치인들
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
		Politicians:   selectedPoliticians, // 기존 사용자의 정치인 목록 또는 기본값
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

// handleClaimInitialCoins는 기존 사용자가 초기 코인을 수동으로 받을 수 있게 해주는 핸들러입니다.
func handleClaimInitialCoins(w http.ResponseWriter, r *http.Request) {
	log.Printf("🎁 초기 코인 지급 요청 시작")
	
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		log.Printf("❌ 사용자 ID를 찾을 수 없음")
		http.Error(w, "사용자 ID를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}
	
	log.Printf("📋 초기 코인 지급 요청 - 사용자: %s", userID)

	// PIN 검증을 위한 요청 바디 파싱
	var reqBody struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식", http.StatusBadRequest)
		return
	}

	// PIN 검증
	log.Printf("🔐 PIN 검증 시작 - 사용자: %s", userID)
	if err := verifyUserPIN(userID, reqBody.PIN); err != nil {
		log.Printf("❌ PIN 검증 실패 - 사용자: %s, 오류: %v", userID, err)
		http.Error(w, "PIN이 올바르지 않습니다", http.StatusUnauthorized)
		return
	}
	log.Printf("✅ PIN 검증 성공 - 사용자: %s", userID)

	// 사용자 계정 조회
	log.Printf("🔍 사용자 계정 조회 시작 - 사용자: %s", userID)
	queryPath := fmt.Sprintf("/account?address=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		log.Printf("❌ ABCI 조회 오류 - 사용자: %s, 오류: %v", userID, err)
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}
	if res.Response.Code != 0 {
		log.Printf("❌ 계정이 블록체인에 없음 - 사용자: %s, 코드: %d, 로그: %s", userID, res.Response.Code, res.Response.Log)
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}
	log.Printf("✅ 사용자 계정 조회 성공 - 사용자: %s", userID)

	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		log.Printf("❌ 계정 정보 파싱 실패 - 사용자: %s, 오류: %v", userID, err)
		http.Error(w, "계정 정보 파싱 실패", http.StatusInternalServerError)
		return
	}
	log.Printf("📋 계정 정보 파싱 성공 - 사용자: %s, InitialSelection: %v, Politicians: %v", userID, account.InitialSelection, account.Politicians)

	// 이미 초기 코인을 받았는지 확인
	if account.InitialSelection {
		log.Printf("❌ 이미 초기 코인을 받은 사용자 - 사용자: %s", userID)
		http.Error(w, "이미 초기 코인을 받으셨습니다", http.StatusBadRequest)
		return
	}
	log.Printf("✅ 초기 코인 지급 가능 - 사용자: %s", userID)

	// 사용자가 선택한 정치인들로 초기 코인 지급 트랜잭션 생성
	userPoliticians := account.Politicians
	if len(userPoliticians) == 0 {
		// 정치인 정보가 없으면 기본 3명 사용
		userPoliticians = []string{"이재명", "윤석열", "이낙연"}
		log.Printf("No politicians found for user %s, using default politicians", userID)
	} else {
		log.Printf("Using user's selected politicians for %s: %v", userID, userPoliticians)
	}

	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	txID := fmt.Sprintf("%s-claim-%d-%x", userID, time.Now().UnixNano(), randBytes)

	txData := ptypes.TxData{
		TxID:        txID,
		Action:      "update_supporters",
		UserID:      userID,
		Politicians: userPoliticians, // 사용자가 선택한 정치인들
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		log.Printf("Error marshaling claim transaction: %v", err)
		http.Error(w, "트랜잭션 생성 실패", http.StatusInternalServerError)
		return
	}

	log.Printf("📡 블록체인 트랜잭션 브로드캐스트 시작 - TxID: %s", txID)
	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		log.Printf("❌ 초기 코인 브로드캐스트 실패 - 사용자: %s, TxID: %s, 오류: %v", userID, txID, err)
		http.Error(w, "초기 코인 지급 실패: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("✅ 블록체인 트랜잭션 성공 - 사용자: %s, TxID: %s", userID, txID)

	totalCoins := len(userPoliticians) * 100
	log.Printf("🎉 초기 코인 지급 성공 - 사용자: %s, 정치인: %v, 총 코인: %d", userID, userPoliticians, totalCoins)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("초기 코인이 성공적으로 지급되었습니다! 각 정치인마다 100개씩 총 %d개의 코인을 받았습니다.", totalCoins),
		"coins_given": totalCoins,
		"politicians": userPoliticians,
	})
}


package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	ptypes "github.com/jclee286/politisian/pkg/types"
)

// handleStablecoinDeposit handles USDT/USDC deposit requests (forwarded from old implementation)
func handleStablecoinDeposit(w http.ResponseWriter, r *http.Request) {
	// This function was moved from wallet_handlers.go - placeholder for now
	http.Error(w, "입금 기능은 현재 업데이트 중입니다", http.StatusServiceUnavailable)
}

// handleStablecoinWithdraw handles USDT/USDC withdrawal requests
func handleStablecoinWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "인증이 필요합니다", http.StatusUnauthorized)
		return
	}

	// 출금 요청 데이터 파싱
	var req ptypes.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "잘못된 요청 형식", http.StatusBadRequest)
		return
	}

	// 입력 데이터 검증
	if req.Amount <= 0 {
		http.Error(w, "출금 금액은 0보다 커야 합니다", http.StatusBadRequest)
		return
	}

	if req.TokenType != "USDT" && req.TokenType != "USDC" {
		http.Error(w, "지원하지 않는 토큰입니다 (USDT 또는 USDC만 가능)", http.StatusBadRequest)
		return
	}

	if req.ToAddress == "" {
		http.Error(w, "출금 주소가 필요합니다", http.StatusBadRequest)
		return
	}

	// PIN 검증
	if err := verifyUserPIN(userID, req.PIN); err != nil {
		http.Error(w, "PIN이 올바르지 않습니다", http.StatusUnauthorized)
		return
	}

	// Polygon 주소 형식 검증
	if !validatePolygonAddress(req.ToAddress) {
		http.Error(w, "올바르지 않은 Polygon 주소 형식입니다", http.StatusBadRequest)
		return
	}

	// 사용자 계정 조회 및 잔액 확인
	account, err := getUserAccount(userID)
	if err != nil {
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	// 토큰별 잔액 확인
	var currentBalance int64
	if req.TokenType == "USDT" {
		currentBalance = account.USDTBalance
	} else {
		currentBalance = account.USDCBalance
	}

	if currentBalance < req.Amount {
		http.Error(w, "출금 가능한 잔액이 부족합니다", http.StatusBadRequest)
		return
	}

	// 최소 출금 금액 확인
	minWithdraw := int64(10000000) // 10 USDT/USDC (6 decimal places)
	if req.Amount < minWithdraw {
		http.Error(w, "최소 출금 금액은 10 USDT/USDC입니다", http.StatusBadRequest)
		return
	}

	log.Printf("💳 %s 출금 요청: 사용자 %s, 금액 %d %s, 주소 %s", req.TokenType, userID, req.Amount, req.TokenType, req.ToAddress)

	// 블록체인에 출금 처리
	txData := ptypes.TxData{
		Action: "withdraw_stablecoin",
		UserID: userID,
		TxID:   fmt.Sprintf("withdraw_%s_%d", userID, time.Now().UnixNano()),
		// 출금 정보를 Politicians 필드에 임시로 전달 (구조 개선 필요)
		Politicians: []string{
			fmt.Sprintf("amount:%d", req.Amount),
			fmt.Sprintf("token_type:%s", req.TokenType),
			fmt.Sprintf("to_address:%s", req.ToAddress),
		},
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		http.Error(w, "출금 처리 중 오류가 발생했습니다", http.StatusInternalServerError)
		return
	}

	// 블록체인에 트랜잭션 전송 (잔액 차감)
	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		log.Printf("❌ 출금 트랜잭션 실패: %v", err)
		http.Error(w, "출금 처리에 실패했습니다", http.StatusInternalServerError)
		return
	}

	// 토큰 주소 결정
	var tokenAddress string
	if req.TokenType == "USDT" {
		tokenAddress = POLYGON_USDT_ADDRESS
	} else {
		tokenAddress = POLYGON_USDC_ADDRESS
	}

	// 실제 Polygon 네트워크로 토큰 전송 (데모에서는 시뮬레이션)
	// TODO: 실제로는 서버의 마스터 지갑에서 사용자가 요청한 주소로 토큰 전송
	txHash, err := sendPolygonTransaction("master_private_key", req.ToAddress, req.Amount, tokenAddress)
	if err != nil {
		log.Printf("❌ Polygon 전송 실패: %v", err)
		// 실패 시 블록체인에서 잔액 복구 필요 (복잡한 롤백 로직)
		http.Error(w, "Polygon 네트워크 전송에 실패했습니다", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ %s 출금 완료: 사용자 %s, 금액 %d %s, Polygon 트랜잭션 %s", req.TokenType, userID, req.Amount, req.TokenType, txHash)

	// 성공 응답
	response := map[string]interface{}{
		"success":    true,
		"message":    "출금이 성공적으로 처리되었습니다",
		"amount":     req.Amount,
		"token_type": req.TokenType,
		"to_address": req.ToAddress,
		"tx_hash":    txHash,
		"status":     "processing",
		"notice":     "Polygon 네트워크 확인까지 약 1-3분이 소요됩니다",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetPolygonAddress returns user's Polygon wallet address
func handleGetPolygonAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "인증이 필요합니다", http.StatusUnauthorized)
		return
	}

	account, err := getUserAccount(userID)
	if err != nil {
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	// 입금 주소가 없으면 새로 생성
	if account.PolygonWalletAddress == "" {
		// 실제 Polygon 지갑 주소 생성
		wallet, err := generatePolygonWallet()
		if err != nil {
			log.Printf("Polygon 지갑 생성 실패: %v", err)
			http.Error(w, "지갑 주소 생성에 실패했습니다", http.StatusInternalServerError)
			return
		}
		
		account.PolygonWalletAddress = wallet.Address
		
		// 블록체인에 업데이트 (실제로는 update_account 액션 필요)
		log.Printf("Generated Polygon deposit address for user %s: %s", userID, account.PolygonWalletAddress)
		
		// TODO: 실제로는 개인키를 안전하게 저장해야 함 (암호화된 형태로)
		// 현재는 입금용 주소만 생성하고 개인키는 저장하지 않음
	}

	response := map[string]interface{}{
		"deposit_address": account.PolygonWalletAddress,
		"network":         "Polygon",
		"supported_tokens": []string{"USDT", "USDC", "MATIC"},
		"usdt_contract":   POLYGON_USDT_ADDRESS,
		"usdc_contract":   POLYGON_USDC_ADDRESS,
		"notice":          "이 주소로 USDT, USDC (Polygon 네트워크)만 보내주세요. 다른 토큰이나 네트워크를 사용하면 자산을 잃을 수 있습니다.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetStablecoinBalance returns user's USDT/USDC balance
func handleGetStablecoinBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "인증이 필요합니다", http.StatusUnauthorized)
		return
	}

	account, err := getUserAccount(userID)
	if err != nil {
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	// 에스크로 동결 금액 고려한 사용 가능한 잔액
	availableUSDT := account.USDTBalance - account.EscrowAccount.FrozenUSDTBalance
	availableUSDC := account.USDCBalance - account.EscrowAccount.FrozenUSDCBalance

	response := map[string]interface{}{
		"usdt_balance":           account.USDTBalance,
		"usdc_balance":           account.USDCBalance,
		"matic_balance":          account.MATICBalance,
		"available_usdt":         availableUSDT,
		"available_usdc":         availableUSDC,
		"frozen_usdt":            account.EscrowAccount.FrozenUSDTBalance,
		"frozen_usdc":            account.EscrowAccount.FrozenUSDCBalance,
		"polygon_wallet_address": account.PolygonWalletAddress,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
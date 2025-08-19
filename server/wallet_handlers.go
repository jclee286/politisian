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

// handleTetherDeposit handles USDT deposit requests
func handleTetherDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "인증이 필요합니다", http.StatusUnauthorized)
		return
	}

	// 입금 요청 데이터 파싱
	var req ptypes.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "잘못된 요청 형식", http.StatusBadRequest)
		return
	}

	// 입력 데이터 검증
	if req.Amount <= 0 {
		http.Error(w, "입금 금액은 0보다 커야 합니다", http.StatusBadRequest)
		return
	}

	if req.TxHash == "" {
		http.Error(w, "트랜잭션 해시가 필요합니다", http.StatusBadRequest)
		return
	}

	if req.FromAddress == "" {
		http.Error(w, "송금 주소가 필요합니다", http.StatusBadRequest)
		return
	}

	// PIN 검증
	if err := verifyUserPIN(userID, req.PIN); err != nil {
		http.Error(w, "PIN이 올바르지 않습니다", http.StatusUnauthorized)
		return
	}

	// TRON 주소 형식 검증
	if !validateTronAddress(req.FromAddress) {
		http.Error(w, "올바르지 않은 TRON 주소 형식입니다", http.StatusBadRequest)
		return
	}

	// 사용자 계정 조회
	account, err := getUserAccount(userID)
	if err != nil {
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	// 입금 주소가 없으면 생성
	if account.TetherWalletAddress == "" {
		wallet, err := generateTronWallet()
		if err != nil {
			log.Printf("TRON 지갑 생성 실패: %v", err)
			http.Error(w, "지갑 주소 생성에 실패했습니다", http.StatusInternalServerError)
			return
		}
		account.TetherWalletAddress = wallet.Address
	}

	// TRON 트랜잭션 검증
	log.Printf("🔍 TRON 트랜잭션 검증 시작: %s", req.TxHash)
	tx, err := verifyTronTransaction(req.TxHash, account.TetherWalletAddress, req.Amount)
	if err != nil {
		log.Printf("❌ 트랜잭션 검증 실패: %v", err)
		http.Error(w, fmt.Sprintf("트랜잭션 검증에 실패했습니다: %v", err), http.StatusBadRequest)
		return
	}

	// 트랜잭션이 확인되면 블록체인에 입금 처리
	txData := ptypes.TxData{
		Action: "deposit_tether",
		UserID: userID,
		TxID:   fmt.Sprintf("deposit_%s_%d", userID, time.Now().UnixNano()),
		// 입금 정보를 Politicians 필드에 임시로 전달 (구조 개선 필요)
		Politicians: []string{
			fmt.Sprintf("amount:%d", req.Amount),
			fmt.Sprintf("tx_hash:%s", req.TxHash),
			fmt.Sprintf("from_address:%s", req.FromAddress),
		},
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		http.Error(w, "입금 처리 중 오류가 발생했습니다", http.StatusInternalServerError)
		return
	}

	// 블록체인에 트랜잭션 전송
	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		log.Printf("❌ 입금 트랜잭션 실패: %v", err)
		http.Error(w, "입금 처리에 실패했습니다", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ USDT 입금 완료: 사용자 %s, 금액 %d USDT, 트랜잭션 %s", userID, req.Amount, req.TxHash)

	// 성공 응답
	response := map[string]interface{}{
		"success": true,
		"message": "입금이 성공적으로 처리되었습니다",
		"amount":  req.Amount,
		"tx_hash": req.TxHash,
		"status":  tx.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTetherWithdraw handles USDT withdrawal requests
func handleTetherWithdraw(w http.ResponseWriter, r *http.Request) {
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

	if req.ToAddress == "" {
		http.Error(w, "출금 주소가 필요합니다", http.StatusBadRequest)
		return
	}

	// PIN 검증
	if err := verifyUserPIN(userID, req.PIN); err != nil {
		http.Error(w, "PIN이 올바르지 않습니다", http.StatusUnauthorized)
		return
	}

	// TRON 주소 형식 검증
	if !validateTronAddress(req.ToAddress) {
		http.Error(w, "올바르지 않은 TRON 주소 형식입니다", http.StatusBadRequest)
		return
	}

	// 사용자 계정 조회 및 잔액 확인
	account, err := getUserAccount(userID)
	if err != nil {
		http.Error(w, "계정을 찾을 수 없습니다", http.StatusNotFound)
		return
	}

	if account.TetherBalance < req.Amount {
		http.Error(w, "출금 가능한 잔액이 부족합니다", http.StatusBadRequest)
		return
	}

	// 최소 출금 금액 확인 (수수료 고려)
	minWithdraw := int64(10000000) // 10 USDT (6 decimal places)
	if req.Amount < minWithdraw {
		http.Error(w, "최소 출금 금액은 10 USDT입니다", http.StatusBadRequest)
		return
	}

	log.Printf("💳 USDT 출금 요청: 사용자 %s, 금액 %d USDT, 주소 %s", userID, req.Amount, req.ToAddress)

	// 블록체인에 출금 처리
	txData := ptypes.TxData{
		Action: "withdraw_tether",
		UserID: userID,
		TxID:   fmt.Sprintf("withdraw_%s_%d", userID, time.Now().UnixNano()),
		// 출금 정보를 Politicians 필드에 임시로 전달 (구조 개선 필요)
		Politicians: []string{
			fmt.Sprintf("amount:%d", req.Amount),
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

	// 실제 TRON 네트워크로 USDT 전송 (데모에서는 시뮬레이션)
	// TODO: 실제로는 서버의 마스터 지갑에서 사용자가 요청한 주소로 USDT 전송
	txHash, err := sendTronTransaction("master_private_key", req.ToAddress, req.Amount)
	if err != nil {
		log.Printf("❌ TRON 전송 실패: %v", err)
		// 실패 시 블록체인에서 잔액 복구 필요 (복잡한 롤백 로직)
		http.Error(w, "TRON 네트워크 전송에 실패했습니다", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ USDT 출금 완료: 사용자 %s, 금액 %d USDT, TRON 트랜잭션 %s", userID, req.Amount, txHash)

	// 성공 응답
	response := map[string]interface{}{
		"success":  true,
		"message":  "출금이 성공적으로 처리되었습니다",
		"amount":   req.Amount,
		"to_address": req.ToAddress,
		"tx_hash":  txHash,
		"status":   "processing",
		"notice":   "TRON 네트워크 확인까지 약 3-10분이 소요됩니다",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
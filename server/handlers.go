package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
)

func handleGetProfileInfo(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok || email == "" {
		http.Error(w, "인증된 사용자 정보를 찾을 수 없습니다.", http.StatusUnauthorized)
		return
	}

	pubKey := ed25519.GenPrivKey().PubKey()
	walletAddress := pubKey.Address().String()

	response := ProfileInfoResponse{
		WalletAddress: walletAddress,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var reqBody ProfileSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식입니다.", http.StatusBadRequest)
		return
	}

	txData := TxData{
		Email:       email,
		Nickname:    reqBody.Nickname,
		Wallet:      reqBody.Wallet,
		Country:     reqBody.Country,
		Gender:      reqBody.Gender,
		BirthYear:   reqBody.BirthYear,
		Politicians: reqBody.Politicians,
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		http.Error(w, "트랜잭션 생성 실패", http.StatusInternalServerError)
		return
	}

	res, err := blockchainClient.BroadcastTxCommit(r.Context(), txBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("블록체인 통신 실패 (RPC 오류): %v", err), http.StatusInternalServerError)
		return
	}

	if res.CheckTx.Code != types.CodeTypeOK {
		errorMsg := fmt.Sprintf("블록체인 트랜잭션 확인 실패: %s (코드: %d)", res.CheckTx.Log, res.CheckTx.Code)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	// Use TxResult for CometBFT v0.38+
	txResult := res.TxResult
	if txResult.Code != types.CodeTypeOK {
		errorMsg := fmt.Sprintf("블록체인 트랜잭션 실행 실패: %s (코드: %d)", txResult.Log, txResult.Code)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("프로필 저장 성공"))
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok || email == "" {
		http.Error(w, "인증된 사용자 정보를 찾을 수 없습니다.", http.StatusUnauthorized)
		return
	}

	res, err := blockchainClient.ABCIQuery(context.Background(), "/account/profile", []byte(email))
	if err != nil {
		http.Error(w, "블록체인에서 대시보드 정보를 가져오는데 실패했습니다.", http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		http.Error(w, "사용자 정보를 조회하는데 실패했습니다.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res.Response.Value)
}

func handleGetPoliticians(w http.ResponseWriter, r *http.Request) {
	res, err := blockchainClient.ABCIQuery(context.Background(), "/politicians/list", nil)
	if err != nil {
		http.Error(w, "블록체인에서 정치인 목록을 가져오는데 실패했습니다.", http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		http.Error(w, "정치인 목록 조회에 실패했습니다.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res.Response.Value)
}

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

	privKey := ed25519.GenPrivKey()
	walletAddress := privKey.PubKey().Address().String()

	responseData := map[string]string{
		"Wallet": walletAddress,
		"Email":  email,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok || email == "" {
		http.Error(w, "인증된 사용자 정보를 찾을 수 없습니다.", http.StatusUnauthorized)
		return
	}

	var profileData struct {
		Wallet      string   `json:"wallet"`
		Country     string   `json:"country"`
		Gender      string   `json:"gender"`
		BirthYear   int      `json:"birthYear"`
		Politicians []string `json:"politicians"`
	}
	if err := json.NewDecoder(r.Body).Decode(&profileData); err != nil {
		http.Error(w, "잘못된 요청 데이터입니다.", http.StatusBadRequest)
		return
	}

	txData := TxData{
		Email:       email,
		Wallet:      profileData.Wallet,
		Nickname:    "NewUser",
		Country:     profileData.Country,
		Gender:      profileData.Gender,
		BirthYear:   profileData.BirthYear,
		Politicians: profileData.Politicians,
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		http.Error(w, "트랜잭션 생성 실패", http.StatusInternalServerError)
		return
	}

	res, err := blockchainClient.BroadcastTxCommit(context.Background(), txBytes)
	if err != nil {
		errorMsg := fmt.Sprintf("블록체인 통신 실패 (RPC 오류): %v", err)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	if res.CheckTx.Code != types.CodeTypeOK {
        errorMsg := fmt.Sprintf("블록체인 트랜잭션 확인 실패: %s (코드: %d)", res.CheckTx.Log, res.CheckTx.Code)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}

	if res.TxResult.Code != types.CodeTypeOK {
        errorMsg := fmt.Sprintf("블록체인 트랜잭션 실행 실패: %s (코드: %d)", res.TxResult.Log, res.TxResult.Code)
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

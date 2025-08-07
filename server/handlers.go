package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	ptypes "politician/pkg/types"
)

func handleGetProfileInfo(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok || email == "" {
		http.Error(w, "인증된 사용자 정보를 찾을 수 없습니다.", http.StatusUnauthorized)
		return
	}

	pubKey := ed25519.GenPrivKey().PubKey()
	walletAddress := pubKey.Address().String()

	response := ptypes.ProfileInfoResponse{
		Wallet: walletAddress,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// broadcastAndCheckTx는 트랜잭션을 브로드캐스트하고 결과를 확인하는 헬퍼 함수입니다.
func broadcastAndCheckTx(ctx context.Context, txBytes []byte) error {
	res, err := blockchainClient.BroadcastTxCommit(ctx, txBytes)
	if err != nil {
		return fmt.Errorf("블록체인 통신 실패 (RPC 오류): %v", err)
	}
	if res.CheckTx.Code != types.CodeTypeOK {
		return fmt.Errorf("블록체인 트랜잭션 확인 실패: %s (코드: %d)", res.CheckTx.Log, res.CheckTx.Code)
	}
	if res.TxResult.Code != types.CodeTypeOK {
		return fmt.Errorf("블록체인 트랜잭션 실행 실패: %s (코드: %d)", res.TxResult.Log, res.TxResult.Code)
	}
	return nil
}

func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var reqBody ptypes.ProfileSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식입니다.", http.StatusBadRequest)
		return
	}

	txData := ptypes.TxData{
		Action:      "create_profile",
		Email:       email,
		Nickname:    reqBody.Nickname,
		Wallet:      reqBody.Wallet,
		Country:     reqBody.Country,
		Gender:      reqBody.Gender,
		BirthYear:   reqBody.BirthYear,
		Politicians: reqBody.Politicians,
		Referrer:    reqBody.Referrer,
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		http.Error(w, "트랜잭션 생성 실패", http.StatusInternalServerError)
		return
	}

	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	res, err := blockchainClient.ABCIQuery(context.Background(), "/account/profile-by-email", []byte(email))
	if err != nil {
		http.Error(w, "블록체인에서 대시보드 정보를 가져오는데 실패했습니다.", http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		http.Error(w, "사용자 정보를 조회하는데 실패했습니다.", http.StatusNotFound)
		return
	}

	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		http.Error(w, "블록체인 데이터 파싱 실패", http.StatusInternalServerError)
		return
	}

	response := ptypes.ProfileInfoResponse{
		Email:           account.Email,
		Nickname:        account.Nickname,
		Wallet:          account.Wallet,
		Country:         account.Country,
		Gender:          account.Gender,
		BirthYear:       account.BirthYear,
		Politicians:     account.Politicians,
		Balance:         account.Balance,
		ReferralCredits: account.ReferralCredits,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "응답 생성 실패", http.StatusInternalServerError)
	}
}

func handleClaimReward(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok || email == "" {
		http.Error(w, "인증된 사용자 정보를 찾을 수 없습니다.", http.StatusUnauthorized)
		return
	}

	// 참고: 보상 요청 시에는 요청자의 지갑 주소를 알아야 하지만,
	// 현재 계정 시스템은 이메일 기반이므로 블록체인에서 조회해야 합니다.
	// 이 부분은 시스템이 지갑 주소 기반으로 전환되면 더 효율적으로 변경될 수 있습니다.

	txData := ptypes.TxData{
		Action: "claim_reward",
		Email:  email, // 블록체인에서 이 이메일로 계정을 찾아야 함
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		http.Error(w, "트랜잭션 생성 실패", http.StatusInternalServerError)
		return
	}

	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("보상 요청 성공"))
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

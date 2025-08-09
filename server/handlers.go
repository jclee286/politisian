package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"crypto/sha256"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	ptypes "politician/pkg/types"
)

func handleGetProfileInfo(w http.ResponseWriter, r *http.Request) {
	// 이제 컨텍스트에서 이메일 대신 지갑 주소를 가져옵니다.
	address, ok := r.Context().Value(userWalletAddressKey).(string)
	if !ok {
		http.Error(w, "컨텍스트에서 지갑 주소를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	// 사용자의 이메일을 기반으로 결정론적(deterministic) 키 쌍을 생성합니다.
	// 이렇게 하면 동일한 이메일에 대해 항상 동일한 지갑 주소가 생성됩니다.
	hasher := sha256.New()
	hasher.Write([]byte(address))
	seed := hasher.Sum(nil)
	
	privKey := ed25519.GenPrivKeyFromSecret(seed)
	pubKey := privKey.PubKey()
	walletAddress := pubKey.Address().String()

	response := ptypes.ProfileInfoResponse{
		Wallet: walletAddress,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// broadcastAndCheckTx는 트랜잭션을 브로드캐스트하고 결과를 확인하는 헬퍼 함수입니다.
func broadcastAndCheckTx(ctx context.Context, txBytes []byte) error {
	// `BroadcastTxCommit`은 타임아웃 위험이 있고, `BroadcastTxAsync`는 결과를 보장하지 않습니다.
	// `BroadcastTxSync`는 트랜잭션이 Mempool에 포함되었는지 확인하여 안정성과 속도의 균형을 맞춥니다.
	// res, err := blockchainClient.BroadcastTxCommit(ctx, txBytes)
	res, err := blockchainClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		return fmt.Errorf("블록체인 통신 실패 (RPC 오류): %v", err)
	}

	if res.Code != types.CodeTypeOK {
		return fmt.Errorf("블록체인 트랜잭션 확인 실패: %s (코드: %d)", res.Log, res.Code)
	}

	return nil
}

func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	// 이제 컨텍스트에서 이메일 대신 지갑 주소를 가져옵니다.
	address, ok := r.Context().Value(userWalletAddressKey).(string)
	if !ok {
		http.Error(w, "컨텍스트에서 지갑 주소를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	var reqBody ptypes.ProfileSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식입니다.", http.StatusBadRequest)
		return
	}

	txData := ptypes.TxData{
		Action:      "create_profile",
		Email:       address,
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
	// 이제 컨텍스트에서 이메일 대신 지갑 주소를 가져옵니다.
	address, ok := r.Context().Value(userWalletAddressKey).(string)
	if !ok {
		http.Error(w, "컨텍스트에서 지갑 주소를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	res, err := blockchainClient.ABCIQuery(context.Background(), "/account/profile-by-email", []byte(address))
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
	// 이제 컨텍스트에서 이메일 대신 지갑 주소를 가져옵니다.
	address, ok := r.Context().Value(userWalletAddressKey).(string)
	if !ok {
		http.Error(w, "컨텍스트에서 지갑 주소를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	// 참고: 보상 요청 시에는 요청자의 지갑 주소를 알아야 하지만,
	// 현재 계정 시스템은 이메일 기반이므로 블록체인에서 조회해야 합니다.
	// 이 부분은 시스템이 지갑 주소 기반으로 전환되면 더 효율적으로 변경될 수 있습니다.

	txData := ptypes.TxData{
		Action: "claim_reward",
		Email:  address, // 블록체인에서 이 이메일로 계정을 찾아야 함
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
	// 이 함수는 모든 사용자에게 동일한 전체 정치인 목록을 반환하므로,
	// 특정 사용자의 지갑 주소를 확인할 필요가 없습니다.
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

func handleProposePolitician(w http.ResponseWriter, r *http.Request) {
	// 이제 컨텍스트에서 이메일 대신 지갑 주소를 가져옵니다.
	address, ok := r.Context().Value(userWalletAddressKey).(string)
	if !ok {
		http.Error(w, "컨텍스트에서 지갑 주소를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	var reqBody ptypes.ProposePoliticianRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식입니다.", http.StatusBadRequest)
		return
	}

	txData := ptypes.TxData{
		Action:         "propose_politician",
		Email:          address,
		PoliticianName: reqBody.Name,
		Region:         reqBody.Region,
		Party:          reqBody.Party,
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

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("정치인 발의가 성공적으로 블록체인에 기록되었습니다."))
}

func handleGetProposals(w http.ResponseWriter, r *http.Request) {
	res, err := blockchainClient.ABCIQuery(context.Background(), "/proposals", nil)
	if err != nil {
		http.Error(w, "블록체인에서 제안 목록을 가져오는데 실패했습니다.", http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		http.Error(w, "제안 목록 조회에 실패했습니다.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res.Response.Value)
}

func handleVoteOnProposal(w http.ResponseWriter, r *http.Request) {
	// 이제 컨텍스트에서 이메일 대신 지갑 주소를 가져옵니다.
	address, ok := r.Context().Value(userWalletAddressKey).(string)
	if !ok {
		http.Error(w, "컨텍스트에서 지갑 주소를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	// URL 경로에서 제안 ID를 추출해야 합니다. 예: /api/proposals/{id}/vote
	// 이 부분은 라우터(예: gorilla/mux)를 사용하면 더 깔끔하게 처리할 수 있습니다.
	// 우선 표준 라이브러리로 처리합니다.
	proposalID := r.URL.Path[len("/api/proposals/") : len(r.URL.Path)-len("/vote")]

	var reqBody ptypes.VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식입니다.", http.StatusBadRequest)
		return
	}

	txData := ptypes.TxData{
		Action:     "vote_on_proposal",
		Email:      address,
		ProposalID: proposalID,
		Vote:       reqBody.Vote,
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
	w.Write([]byte("투표가 성공적으로 블록체인에 기록되었습니다."))
}

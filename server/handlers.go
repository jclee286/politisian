package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	ptypes "politisian/pkg/types"

	"github.com/cometbft/cometbft/abci/types"
)

// broadcastAndCheckTx는 트랜잭션을 브로드캐스트하고 결과를 확인하는 헬퍼 함수입니다.
func broadcastAndCheckTx(ctx context.Context, txBytes []byte) error {
	res, err := blockchainClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		return fmt.Errorf("블록체인 통신 실패 (RPC 오류): %v", err)
	}

	if res.Code != types.CodeTypeOK {
		return fmt.Errorf("블록체인 트랜잭션 확인 실패: %s (코드: %d)", res.Log, res.Code)
	}

	return nil
}

// handleProfileSave는 사용자의 프로필(지지 정치인)을 저장하는 요청을 처리합니다.
func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		http.Error(w, "사용자 ID를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	var reqBody struct {
		Politisian []string `json:"politisian"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식입니다.", http.StatusBadRequest)
		return
	}

	txData := ptypes.TxData{
		Action:     "update_supporters", // 이 액션은 ABCI 앱에서 처리해야 합니다.
		UserID:     userID,
		Politisian: reqBody.Politisian,
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
	json.NewEncoder(w).Encode(map[string]string{"log": "프로필이 성공적으로 저장되었습니다."})
}


// handleUserProfile는 사용자의 프로필 정보를 반환합니다.
func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		http.Error(w, "사용자 ID를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	// ABCI 쿼리를 통해 사용자 계정 정보 가져오기
	queryPath := fmt.Sprintf("/account?address=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		http.Error(w, "프로필 정보 조회 실패", http.StatusInternalServerError)
		return
	}
	if res.Response.Code != 0 {
		http.Error(w, "계정을 찾을 수 없습니다.", http.StatusNotFound)
		return
	}

	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		http.Error(w, "프로필 정보 파싱 실패", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}


// handleGetPolitisians는 등록된 전체 정치인 목록을 반환합니다.
func handleGetPolitisians(w http.ResponseWriter, r *http.Request) {
	res, err := blockchainClient.ABCIQuery(context.Background(), "/politisian/list", nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("블록체인 쿼리 실패: %v", err), http.StatusInternalServerError)
		return
	}

	if res.Response.Code != 0 {
		http.Error(w, "정치인 목록 조회에 실패했습니다.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res.Response.Value)
}

// handleProposePolitisian는 새로운 정치인을 등록 제안하는 요청을 처리합니다.
func handleProposePolitisian(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		http.Error(w, "사용자 ID를 찾을 수 없습니다.", http.StatusInternalServerError)
		return
	}

	var reqBody struct {
		Name   string `json:"name"`
		Region string `json:"region"`
		Party  string `json:"party"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청 형식입니다.", http.StatusBadRequest)
		return
	}

	txData := ptypes.TxData{
		Action:         "propose_politisian",
		UserID:         userID,
		PolitisianName: reqBody.Name,
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
	json.NewEncoder(w).Encode(map[string]string{"log": "정치인 발의가 성공적으로 블록체인에 기록되었습니다."})
}

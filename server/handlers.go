package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
		http.Error(w, "프로필 정보 조회 실패", http.StatusInternalServerError)
		return
	}
	if res.Response.Code != 0 {
		http.Error(w, "계정을 찾을 수 없습니다.", http.StatusNotFound)
		return
	}

	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		log.Printf("Error unmarshalling user profile: %v", err)
		http.Error(w, "프로필 정보 파싱 실패", http.StatusInternalServerError)
		return
	}
	
	log.Printf("Successfully fetched and sending profile for user %s", userID)
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

// handleProfileSave는 사용자의 프로필(지지 정치인)을 저장하는 요청을 처리합니다.
func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to handle /api/profile/save request")
	userID, _ := r.Context().Value("userID").(string)
	var reqBody struct {
		Politicians []string `json:"politicians"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "잘못된 요청", http.StatusBadRequest)
		return
	}
	log.Printf("User %s is saving profile with politicians: %v", userID, reqBody.Politicians)

	txData := ptypes.TxData{
		Action:      "update_supporters",
		UserID:      userID,
		Politicians: reqBody.Politicians,
	}
	txBytes, _ := json.Marshal(txData)

	if err := broadcastAndCheckTx(r.Context(), txBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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

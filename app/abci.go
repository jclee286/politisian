package app

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cometbft/cometbft/abci/types"
	ptypes "politisian/pkg/types"
)

// Info는 애플리케이션의 상태 정보를 반환합니다.
func (app *PoliticianApp) Info(req *types.RequestInfo) (*types.ResponseInfo, error) {
	return &types.ResponseInfo{}, nil
}

// Query는 애플리케이션의 상태를 조회합니다.
func (app *PoliticianApp) Query(req *types.RequestQuery) (*types.ResponseQuery, error) {
	log.Printf("[ABCI] Query: 경로 '%s'에 대한 쿼리 수신", req.Path)
	switch req.Path {
	case "/politisian/list":
		res, err := json.Marshal(app.politicians)
		if err != nil {
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal politicians list"}, nil
		}
		return &types.ResponseQuery{Value: res}, nil
	default:
		return &types.ResponseQuery{Code: 1, Log: "unknown query path"}, nil
	}
}

// DeliverTx는 트랜잭션을 처리하여 애플리케이션 상태를 업데이트합니다.
func (app *PoliticianApp) DeliverTx(req *types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	var txData ptypes.TxData
	if err := json.Unmarshal(req.Tx, &txData); err != nil {
		return &types.ResponseDeliverTx{Code: 1, Log: "failed to parse transaction data"}, nil
	}

	log.Printf("[ABCI] DeliverTx: 사용자 '%s'로부터 액션 '%s' 처리 중", txData.UserID, txData.Action)

	switch txData.Action {
	case "update_supporters":
		return app.updateSupporters(&txData), nil
	case "propose_politisian":
		return app.proposePolitician(&txData), nil
	default:
		return &types.ResponseDeliverTx{Code: 10, Log: "unknown action"}, nil
	}
}

// Commit은 상태 변경사항을 데이터베이스에 영구 저장합니다.
func (app *PoliticianApp) Commit() (*types.ResponseCommit, error) {
	if err := app.saveState(); err != nil {
		log.Printf("심각한 오류: 상태 저장 실패: %v", err)
	}
	return &types.ResponseCommit{}, nil
}

// proposePolitician는 새로운 정치인을 제안하는 트랜잭션을 처리합니다.
func (app *PoliticianApp) proposePolitician(txData *ptypes.TxData) *types.ExecTxResult {
	if txData.PoliticianName == "" || txData.Region == "" || txData.Party == "" {
		return &types.ExecTxResult{Code: 20, Log: "politician name, region, and party are required"}
	}

	if _, exists := app.politicians[txData.PoliticianName]; exists {
		return &types.ExecTxResult{Code: 21, Log: "politician with this name already exists"}
	}

	newPolitician := &ptypes.Politician{
		Name:         txData.PoliticianName,
		Region:       txData.Region,
		Party:        txData.Party,
		Supporters:   []string{txData.UserID},
		TokensMinted: 0,
		MaxTokens:    1000000,
	}

	app.politicians[newPolitician.Name] = newPolitician
	log.Printf("[ABCI] 새로운 정치인 '%s'가 사용자 '%s'에 의해 제안되었습니다.", newPolitician.Name, txData.UserID)

	return &types.ExecTxResult{Code: types.CodeTypeOK, Log: "politician proposed successfully"}
}

// updateSupporters는 사용자가 지지하는 정치인 목록을 업데이트합니다.
func (app *PoliticianApp) updateSupporters(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		account = &ptypes.Account{
			Address:     txData.UserID,
			Politicians: []string{},
		}
		app.accounts[txData.UserID] = account
		log.Printf("[ABCI] 새로운 계정 생성: %s", txData.UserID)
	}

	account.Politicians = txData.Politicians
	app.accounts[txData.UserID] = account

	log.Printf("[ABCI] 사용자 '%s'의 지지 정치인 목록이 업데이트되었습니다: %v", txData.UserID, txData.Politicians)
	return &types.ExecTxResult{Code: types.CodeTypeOK, Log: "supporters updated successfully"}
}

// CheckTx는 트랜잭션이 유효한지 검사합니다.
func (app *PoliticianApp) CheckTx(req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

// InitChain는 체인이 처음 시작될 때 호출됩니다. 제네시스 상태를 초기화합니다.
func (app *PoliticianApp) InitChain(req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	log.Println("[ABCI] InitChain: 체인 초기화 시작...")
	var genesisState ptypes.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, fmt.Errorf("failed to parse genesis state: %w", err)
	}

	if genesisState.Politicians != nil {
		app.politicians = genesisState.Politicians
		log.Printf("[ABCI] InitChain: 제네시스에서 %d명의 정치인 정보를 로드했습니다.", len(app.politicians))
	}
	if genesisState.Accounts != nil {
		app.accounts = genesisState.Accounts
		log.Printf("[ABCI] InitChain: 제네시스에서 %d개의 계정 정보를 로드했습니다.", len(app.accounts))
	}

	return &types.ResponseInitChain{}, nil
}

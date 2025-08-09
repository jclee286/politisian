package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cometbft/cometbft/abci/types"
	ptypes "politisian/pkg/types"
)

// Info는 애플리케이션의 상태 정보를 반환합니다.
func (app *PolitisianApp) Info(req *types.RequestInfo) (*types.ResponseInfo, error) {
	return &types.ResponseInfo{}, nil
}

// Query는 애플리케이션의 상태를 조회합니다.
func (app *PolitisianApp) Query(req *types.RequestQuery) (*types.ResponseQuery, error) {
	log.Printf("[ABCI] Query: 경로 '%s'에 대한 쿼리 수신", req.Path)
	switch req.Path {
	case "/politisian/list":
		res, err := json.Marshal(app.politisian)
		if err != nil {
			log.Printf("[ABCI] Error: failed to marshal politisian list: %v", err)
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal politisian list"}, nil
		}
		return &types.ResponseQuery{Value: res}, nil

	case "/state": // 전체 상태를 조회 (디버깅용)
		appState := AppState{
			Accounts:   app.accounts,
			Proposals:  app.proposals,
			Politisian: app.politisian,
		}
		res, err := json.Marshal(appState)
		if err != nil {
			return &types.ResponseQuery{Code: 5, Log: "failed to marshal full state"}, nil
		}
		return &types.ResponseQuery{Value: res}, nil

	default:
		return &types.ResponseQuery{Code: 1, Log: "unknown query path"}, nil
	}
}

// DeliverTx는 트랜잭션을 처리하여 애플리케이션 상태를 업데이트합니다.
func (app *PolitisianApp) DeliverTx(req *types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	var txData ptypes.TxData
	if err := json.Unmarshal(req.Tx, &txData); err != nil {
		log.Printf("[ABCI] DeliverTx Error: 트랜잭션 파싱 실패: %v", err)
		return &types.ResponseDeliverTx{Code: 1, Log: "failed to parse transaction data"}, nil
	}

	log.Printf("[ABCI] DeliverTx: 사용자 '%s'로부터 액션 '%s' 처리 중", txData.UserID, txData.Action)

	switch txData.Action {
	case "update_supporters":
		return app.updateSupporters(&txData)
	case "propose_politisian":
		return app.proposePolitisian(&txData)
	default:
		return &types.ResponseDeliverTx{Code: 10, Log: "unknown action"}, nil
	}
}

// Commit은 상태 변경사항을 데이터베이스에 영구 저장합니다.
func (app *PolitisianApp) Commit() (*types.ResponseCommit, error) {
	if err := app.saveState(); err != nil {
		log.Printf("심각한 오류: 상태 저장 실패: %v", err)
		// 실제 운영 환경에서는 여기서 패닉을 발생시켜 노드를 중지시킬 수 있습니다.
	}
	return &types.ResponseCommit{}, nil
}

// proposePolitisian는 새로운 정치인을 제안하는 트랜잭션을 처리합니다.
func (app *PolitisianApp) proposePolitisian(txData *ptypes.TxData) (*types.ResponseDeliverTx, error) {
	if txData.PoliticianName == "" || txData.Region == "" || txData.Party == "" {
		return &types.ResponseDeliverTx{Code: 20, Log: "politician name, region, and party are required"}, nil
	}

	if _, exists := app.politisian[txData.PoliticianName]; exists {
		return &types.ResponseDeliverTx{Code: 21, Log: "politisian with this name already exists"}, nil
	}

	newPolitisian := &ptypes.Politician{
		Name:         txData.PoliticianName,
		Region:       txData.Region,
		Party:        txData.Party,
		Supporters:   []string{txData.UserID},
		TokensMinted: 0,
		MaxTokens:    1000000,
	}

	app.politisian[newPolitisian.Name] = newPolitisian
	log.Printf("[ABCI] 새로운 정치인 '%s'가 사용자 '%s'에 의해 제안되었습니다.", newPolitisian.Name, txData.UserID)

	return &types.ResponseDeliverTx{Code: types.CodeTypeOK, Log: "politisian proposed successfully"}, nil
}

// updateSupporters는 사용자가 지지하는 정치인 목록을 업데이트합니다.
func (app *PolitisianApp) updateSupporters(txData *ptypes.TxData) (*types.ResponseDeliverTx, error) {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		// 계정이 없으면 새로 생성
		account = &ptypes.Account{
			Address:    txData.UserID,
			Politisian: []string{},
		}
		app.accounts[txData.UserID] = account
		log.Printf("[ABCI] 새로운 계정 생성: %s", txData.UserID)
	}

	account.Politisian = txData.Politisian
	app.accounts[txData.UserID] = account

	log.Printf("[ABCI] 사용자 '%s'의 지지 정치인 목록이 업데이트되었습니다: %v", txData.UserID, txData.Politisian)
	return &types.ResponseDeliverTx{Code: types.CodeTypeOK, Log: "supporters updated successfully"}, nil
}

// CheckTx는 트랜잭션이 유효한지 검사합니다.
func (app *PolitisianApp) CheckTx(req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

// InitChain는 체인이 처음 시작될 때 호출됩니다. 제네시스 상태를 초기화합니다.
func (app *PolitisianApp) InitChain(req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	log.Println("[ABCI] InitChain: 체인 초기화 시작...")
	var genesisState ptypes.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, fmt.Errorf("failed to parse genesis state: %w", err)
	}

	if genesisState.Politisian != nil {
		app.politisian = genesisState.Politisian
		log.Printf("[ABCI] InitChain: 제네시스에서 %d명의 정치인 정보를 로드했습니다.", len(app.politisian))
	}
	if genesisState.Accounts != nil {
		app.accounts = genesisState.Accounts
		log.Printf("[ABCI] InitChain: 제네시스에서 %d개의 계정 정보를 로드했습니다.", len(app.accounts))
	}

	return &types.ResponseInitChain{}, nil
}

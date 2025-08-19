package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	ptypes "github.com/jclee286/politisian/pkg/types"
)

// handleGetPoliticianPrices는 정치인 코인 가격 순위를 반환합니다.
func handleGetPoliticianPrices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 모든 정치인의 가격 정보 수집
	prices, err := getAllPoliticianPrices()
	if err != nil {
		log.Printf("Error getting politician prices: %v", err)
		http.Error(w, "가격 정보를 불러올 수 없습니다", http.StatusInternalServerError)
		return
	}

	// 가격순으로 정렬
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].CurrentPrice > prices[j].CurrentPrice
	})

	// 순위 설정
	for i := range prices {
		prices[i].Rank = i + 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prices)
}

// handleGetOrderBook은 특정 정치인의 오더북을 반환합니다.
func handleGetOrderBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// URL에서 정치인 ID 추출
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "정치인 ID가 필요합니다", http.StatusBadRequest)
		return
	}
	politicianID := parts[3] // /api/orderbook/{politician_id}

	orderBook, err := getOrderBookForPolitician(politicianID)
	if err != nil {
		log.Printf("Error getting orderbook for %s: %v", politicianID, err)
		http.Error(w, "오더북을 불러올 수 없습니다", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orderBook)
}

// handlePlaceOrder는 거래 주문을 처리합니다.
func handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "인증이 필요합니다", http.StatusUnauthorized)
		return
	}

	var req ptypes.TradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "잘못된 요청 형식", http.StatusBadRequest)
		return
	}

	// 입력 검증
	if req.PoliticianID == "" || req.OrderType == "" || req.Quantity <= 0 || req.Price <= 0 {
		http.Error(w, "모든 필드를 올바르게 입력해주세요", http.StatusBadRequest)
		return
	}

	if req.OrderType != "buy" && req.OrderType != "sell" {
		http.Error(w, "주문 타입은 buy 또는 sell이어야 합니다", http.StatusBadRequest)
		return
	}

	// Currency 기본값 설정
	if req.Currency == "" {
		req.Currency = "USDT" // 기본값을 USDT로 설정
	}
	
	if req.Currency != "USDT" && req.Currency != "USDC" {
		http.Error(w, "통화는 USDT 또는 USDC여야 합니다", http.StatusBadRequest)
		return
	}

	// PIN 검증
	if err := verifyUserPIN(userID, req.PIN); err != nil {
		http.Error(w, "PIN이 올바르지 않습니다", http.StatusUnauthorized)
		return
	}

	// 주문 처리
	orderID, err := placeTradeOrder(userID, req)
	if err != nil {
		log.Printf("Error placing order: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"message":  "주문이 성공적으로 등록되었습니다",
		"order_id": orderID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCancelOrder는 주문 취소를 처리합니다.
func handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "인증이 필요합니다", http.StatusUnauthorized)
		return
	}

	// URL에서 주문 ID 추출
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "주문 ID가 필요합니다", http.StatusBadRequest)
		return
	}
	orderID := parts[3] // /api/cancel-order/{order_id}

	if err := cancelTradeOrder(userID, orderID); err != nil {
		log.Printf("Error cancelling order: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "주문이 취소되었습니다",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetUserOrders는 사용자의 활성 주문 목록을 반환합니다.
func handleGetUserOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "인증이 필요합니다", http.StatusUnauthorized)
		return
	}

	orders, err := getUserActiveOrders(userID)
	if err != nil {
		log.Printf("Error getting user orders: %v", err)
		http.Error(w, "주문 목록을 불러올 수 없습니다", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// 헬퍼 함수들

// getAllPoliticianPrices는 모든 정치인의 가격 정보를 수집합니다.
func getAllPoliticianPrices() ([]ptypes.PoliticianPrice, error) {
	// 등록된 정치인 목록 조회
	queryPath := "/politicians"
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		return nil, fmt.Errorf("politicians query error: %v", err)
	}

	if res.Response.Code != 0 {
		return nil, fmt.Errorf("politicians not found")
	}

	var politicians map[string]*ptypes.Politician
	if err := json.Unmarshal(res.Response.Value, &politicians); err != nil {
		return nil, fmt.Errorf("politicians unmarshal error: %v", err)
	}

	var prices []ptypes.PoliticianPrice

	// 각 정치인의 가격 정보 계산
	for id, politician := range politicians {
		orderBook, err := getOrderBookForPolitician(id)
		if err != nil {
			log.Printf("Error getting orderbook for %s: %v", id, err)
			continue
		}

		// 현재 가격 계산 (최근 체결가 또는 중간가)
		currentPrice := calculateCurrentPrice(orderBook)

		price := ptypes.PoliticianPrice{
			PoliticianID: id,
			Name:         politician.Name,
			CurrentPrice: currentPrice,
			Change24h:    0, // TODO: 24시간 변동가 계산
			Volume24h:    orderBook.Volume24h,
		}

		prices = append(prices, price)
	}

	return prices, nil
}

// getOrderBookForPolitician은 특정 정치인의 오더북을 반환합니다.
func getOrderBookForPolitician(politicianID string) (*ptypes.OrderBook, error) {
	// 블록체인에서 해당 정치인의 모든 활성 주문 조회
	queryPath := fmt.Sprintf("/orders?politician_id=%s", politicianID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		return nil, fmt.Errorf("orders query error: %v", err)
	}

	var orders []ptypes.TradeOrder
	if res.Response.Code == 0 {
		if err := json.Unmarshal(res.Response.Value, &orders); err != nil {
			return nil, fmt.Errorf("orders unmarshal error: %v", err)
		}
	}

	// 매수/매도 주문 분리 및 정렬
	var buyOrders, sellOrders []ptypes.TradeOrder

	for _, order := range orders {
		if order.Status == "active" {
			if order.OrderType == "buy" {
				buyOrders = append(buyOrders, order)
			} else if order.OrderType == "sell" {
				sellOrders = append(sellOrders, order)
			}
		}
	}

	// 매수 주문: 가격 높은 순 정렬
	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].Price > buyOrders[j].Price
	})

	// 매도 주문: 가격 낮은 순 정렬
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].Price < sellOrders[j].Price
	})

	orderBook := &ptypes.OrderBook{
		PoliticianID: politicianID,
		BuyOrders:    buyOrders,
		SellOrders:   sellOrders,
		LastPrice:    calculateLastPrice(politicianID),
		Volume24h:    calculateVolume24h(politicianID),
	}

	return orderBook, nil
}

// calculateCurrentPrice는 현재 가격을 계산합니다.
func calculateCurrentPrice(orderBook *ptypes.OrderBook) int64 {
	// 최근 체결가가 있으면 사용
	if orderBook.LastPrice > 0 {
		return orderBook.LastPrice
	}

	// 없으면 매수 1호가와 매도 1호가의 평균
	var buyPrice, sellPrice int64

	if len(orderBook.BuyOrders) > 0 {
		buyPrice = orderBook.BuyOrders[0].Price
	}

	if len(orderBook.SellOrders) > 0 {
		sellPrice = orderBook.SellOrders[0].Price
	}

	if buyPrice > 0 && sellPrice > 0 {
		return (buyPrice + sellPrice) / 2
	}

	if buyPrice > 0 {
		return buyPrice
	}

	if sellPrice > 0 {
		return sellPrice
	}

	// 기본 가격
	return 1000
}

// calculateLastPrice는 최근 체결가를 계산합니다.
func calculateLastPrice(politicianID string) int64 {
	// TODO: 실제 거래 기록에서 최근 체결가 조회
	return 0
}

// calculateVolume24h는 24시간 거래량을 계산합니다.
func calculateVolume24h(politicianID string) int64 {
	// TODO: 실제 거래 기록에서 24시간 거래량 계산
	return 0
}

// placeTradeOrder는 거래 주문을 처리합니다.
func placeTradeOrder(userID string, req ptypes.TradeRequest) (string, error) {
	// 사용자 계정 조회
	account, err := getUserAccount(userID)
	if err != nil {
		return "", fmt.Errorf("계정을 찾을 수 없습니다")
	}

	// 에스크로 동결 금액 계산
	var escrowAmount int64
	var availableBalance int64
	
	if req.OrderType == "buy" {
		// 매수: 스테이블코인 동결
		escrowAmount = req.Quantity * req.Price
		if req.Currency == "USDT" {
			availableBalance = account.USDTBalance - account.EscrowAccount.FrozenUSDTBalance
		} else {
			availableBalance = account.USDCBalance - account.EscrowAccount.FrozenUSDCBalance
		}
		
		if availableBalance < escrowAmount {
			return "", fmt.Errorf("사용 가능한 %s이 부족합니다 (필요: %d, 사용가능: %d)", req.Currency, escrowAmount, availableBalance)
		}
	} else {
		// 매도: 정치인 코인 동결
		escrowAmount = req.Quantity
		frozenCoins := account.EscrowAccount.FrozenPoliticianCoins[req.PoliticianID]
		availableBalance = account.PoliticianCoins[req.PoliticianID] - frozenCoins
		
		if availableBalance < escrowAmount {
			return "", fmt.Errorf("사용 가능한 정치인 코인이 부족합니다 (필요: %d, 사용가능: %d)", escrowAmount, availableBalance)
		}
	}

	// 주문 ID 생성
	orderID := fmt.Sprintf("order_%s_%d", userID, time.Now().UnixNano())

	// 주문 데이터 생성
	order := ptypes.TradeOrder{
		ID:             orderID,
		UserID:         userID,
		PoliticianID:   req.PoliticianID,
		OrderType:      req.OrderType,
		Currency:       req.Currency,
		Quantity:       req.Quantity,
		Price:          req.Price,
		Status:         "active",
		FilledQuantity: 0,
		EscrowAmount:   escrowAmount,
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	// 블록체인에 주문 추가
	txData := ptypes.TxData{
		Action: "place_order",
		UserID: userID,
	}

	orderBytes, err := json.Marshal(order)
	if err != nil {
		return "", fmt.Errorf("order marshal error: %v", err)
	}

	txData.TxID = orderID
	// 주문 데이터를 TxData에 포함 (임시로 Politicians 필드 사용)
	txData.Politicians = []string{string(orderBytes)}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		return "", fmt.Errorf("transaction marshal error: %v", err)
	}

	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		return "", fmt.Errorf("blockchain transaction error: %v", err)
	}

	// 에스크로 동결 처리
	if err := freezeEscrowAmount(userID, &order); err != nil {
		log.Printf("⚠️ 에스크로 동결 실패, 주문은 등록됨: %v", err)
		// 에스크로 동결 실패 시에도 주문은 유지되지만 경고 로그
	}

	// 매칭 시도 (비동기)
	go tryMatchOrders(req.PoliticianID)

	return orderID, nil
}

// cancelTradeOrder는 거래 주문을 취소합니다.
func cancelTradeOrder(userID, orderID string) error {
	// 주문 소유권 확인
	order, err := getTradeOrder(orderID)
	if err != nil {
		return fmt.Errorf("주문을 찾을 수 없습니다")
	}

	if order.UserID != userID {
		return fmt.Errorf("주문을 취소할 권한이 없습니다")
	}

	if order.Status != "active" {
		return fmt.Errorf("이미 처리된 주문입니다")
	}

	// 주문 취소 트랜잭션
	txData := ptypes.TxData{
		Action: "cancel_order",
		UserID: userID,
		TxID:   fmt.Sprintf("cancel_%s_%d", orderID, time.Now().UnixNano()),
		Politicians: []string{orderID}, // 주문 ID 전달
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("transaction marshal error: %v", err)
	}

	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		return err
	}

	// 에스크로 해제 처리
	if err := releaseEscrowAmount(userID, orderID); err != nil {
		log.Printf("⚠️ 에스크로 해제 실패: %v", err)
		// 에스크로 해제 실패 시에도 주문 취소는 완료됨
	}

	return nil
}

// getUserActiveOrders는 사용자의 활성 주문을 반환합니다.
func getUserActiveOrders(userID string) ([]ptypes.TradeOrder, error) {
	queryPath := fmt.Sprintf("/user-orders?user_id=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		return nil, fmt.Errorf("user orders query error: %v", err)
	}

	var orders []ptypes.TradeOrder
	if res.Response.Code == 0 {
		if err := json.Unmarshal(res.Response.Value, &orders); err != nil {
			return nil, fmt.Errorf("orders unmarshal error: %v", err)
		}
	}

	// 활성 주문만 필터링
	var activeOrders []ptypes.TradeOrder
	for _, order := range orders {
		if order.Status == "active" {
			activeOrders = append(activeOrders, order)
		}
	}

	return activeOrders, nil
}

// getTradeOrder는 특정 주문을 조회합니다.
func getTradeOrder(orderID string) (*ptypes.TradeOrder, error) {
	queryPath := fmt.Sprintf("/order?id=%s", orderID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		return nil, fmt.Errorf("order query error: %v", err)
	}

	if res.Response.Code != 0 {
		return nil, fmt.Errorf("order not found")
	}

	var order ptypes.TradeOrder
	if err := json.Unmarshal(res.Response.Value, &order); err != nil {
		return nil, fmt.Errorf("order unmarshal error: %v", err)
	}

	return &order, nil
}

// getUserAccount는 사용자 계정을 조회합니다.
func getUserAccount(userID string) (*ptypes.Account, error) {
	queryPath := fmt.Sprintf("/account?address=%s", userID)
	res, err := blockchainClient.ABCIQuery(context.Background(), queryPath, nil)
	if err != nil {
		return nil, fmt.Errorf("account query error: %v", err)
	}

	if res.Response.Code != 0 {
		return nil, fmt.Errorf("account not found")
	}

	var account ptypes.Account
	if err := json.Unmarshal(res.Response.Value, &account); err != nil {
		return nil, fmt.Errorf("account unmarshal error: %v", err)
	}

	return &account, nil
}

// verifyUserPIN은 사용자 PIN을 검증합니다.
func verifyUserPIN(userID, pin string) error {
	// 기존 PIN 검증 로직 재사용
	// TODO: 실제 PIN 검증 구현
	return nil
}

// tryMatchOrders는 주문 매칭을 시도합니다.
func tryMatchOrders(politicianID string) {
	log.Printf("🔄 주문 매칭 시작: %s", politicianID)
	
	// 해당 정치인의 오더북 조회
	orderBook, err := getOrderBookForPolitician(politicianID)
	if err != nil {
		log.Printf("❌ 오더북 조회 실패: %v", err)
		return
	}
	
	// 매수/매도 주문이 모두 있는지 확인
	if len(orderBook.BuyOrders) == 0 || len(orderBook.SellOrders) == 0 {
		log.Printf("📝 매칭할 주문 없음 - 매수: %d, 매도: %d", len(orderBook.BuyOrders), len(orderBook.SellOrders))
		return
	}
	
	// 매수 최고가와 매도 최저가 비교
	for len(orderBook.BuyOrders) > 0 && len(orderBook.SellOrders) > 0 {
		buyOrder := &orderBook.BuyOrders[0]   // 가장 높은 매수 가격
		sellOrder := &orderBook.SellOrders[0] // 가장 낮은 매도 가격
		
		// 가격이 맞지 않으면 매칭 중단
		if buyOrder.Price < sellOrder.Price {
			log.Printf("💸 가격 불일치 - 매수: %d, 매도: %d", buyOrder.Price, sellOrder.Price)
			break
		}
		
		// 거래 체결
		if err := executeTrade(buyOrder, sellOrder); err != nil {
			log.Printf("❌ 거래 체결 실패: %v", err)
			break
		}
		
		// 주문 상태 업데이트 후 오더북 재조회
		orderBook, err = getOrderBookForPolitician(politicianID)
		if err != nil {
			log.Printf("❌ 오더북 재조회 실패: %v", err)
			break
		}
	}
	
	log.Printf("✅ 주문 매칭 완료: %s", politicianID)
}

// executeTrade는 실제 거래를 체결합니다.
func executeTrade(buyOrder, sellOrder *ptypes.TradeOrder) error {
	log.Printf("🤝 거래 체결 시작 - 매수주문: %s, 매도주문: %s", buyOrder.ID, sellOrder.ID)
	
	// 체결 수량 결정 (둘 중 작은 값)
	tradeQuantity := buyOrder.Quantity - buyOrder.FilledQuantity
	remainingSellQuantity := sellOrder.Quantity - sellOrder.FilledQuantity
	
	if remainingSellQuantity < tradeQuantity {
		tradeQuantity = remainingSellQuantity
	}
	
	// 체결 가격 결정 (먼저 등록된 주문의 가격 적용)
	var tradePrice int64
	if buyOrder.CreatedAt < sellOrder.CreatedAt {
		tradePrice = buyOrder.Price
	} else {
		tradePrice = sellOrder.Price
	}
	
	totalAmount := tradeQuantity * tradePrice
	
	log.Printf("📊 거래 세부사항 - 수량: %d, 가격: %d, 총액: %d", tradeQuantity, tradePrice, totalAmount)
	
	// 거래 기록 생성
	tradeID := fmt.Sprintf("trade_%d_%s", time.Now().UnixNano(), buyOrder.PoliticianID)
	trade := ptypes.Trade{
		ID:           tradeID,
		BuyOrderID:   buyOrder.ID,
		SellOrderID:  sellOrder.ID,
		BuyerID:      buyOrder.UserID,
		SellerID:     sellOrder.UserID,
		PoliticianID: buyOrder.PoliticianID,
		Quantity:     tradeQuantity,
		Price:        tradePrice,
		TotalAmount:  totalAmount,
		Timestamp:    time.Now().Unix(),
		Status:       "processing",
	}
	
	// 블록체인에 거래 체결 트랜잭션 전송
	txData := ptypes.TxData{
		Action: "execute_trade",
		TxID:   tradeID,
	}
	
	// 거래 데이터를 JSON으로 직렬화하여 전송
	tradeBytes, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("거래 데이터 직렬화 실패: %v", err)
	}
	
	txData.Politicians = []string{string(tradeBytes)}
	
	txBytes, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("트랜잭션 직렬화 실패: %v", err)
	}
	
	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		return fmt.Errorf("거래 체결 트랜잭션 실패: %v", err)
	}
	
	log.Printf("✅ 거래 체결 완료: %s", tradeID)
	return nil
}

// freezeEscrowAmount는 에스크로 금액을 동결합니다.
func freezeEscrowAmount(userID string, order *ptypes.TradeOrder) error {
	log.Printf("🔒 에스크로 동결 시작 - 사용자: %s, 주문: %s", userID, order.ID)
	
	// 에스크로 동결 트랜잭션 생성
	txData := ptypes.TxData{
		Action: "freeze_escrow",
		UserID: userID,
		TxID:   fmt.Sprintf("freeze_%s_%d", order.ID, time.Now().UnixNano()),
	}
	
	// 주문 정보를 JSON으로 직렬화
	orderBytes, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("주문 데이터 직렬화 실패: %v", err)
	}
	
	txData.Politicians = []string{string(orderBytes)}
	
	txBytes, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("트랜잭션 직렬화 실패: %v", err)
	}
	
	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		return fmt.Errorf("에스크로 동결 트랜잭션 실패: %v", err)
	}
	
	log.Printf("✅ 에스크로 동결 완료 - 사용자: %s, 금액: %d", userID, order.EscrowAmount)
	return nil
}

// releaseEscrowAmount는 에스크로 금액을 해제합니다.
func releaseEscrowAmount(userID, orderID string) error {
	log.Printf("🔓 에스크로 해제 시작 - 사용자: %s, 주문: %s", userID, orderID)
	
	txData := ptypes.TxData{
		Action: "release_escrow",
		UserID: userID,
		TxID:   fmt.Sprintf("release_%s_%d", orderID, time.Now().UnixNano()),
		Politicians: []string{orderID}, // 주문 ID 전달
	}
	
	txBytes, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("트랜잭션 직렬화 실패: %v", err)
	}
	
	if err := broadcastAndCheckTx(context.Background(), txBytes); err != nil {
		return fmt.Errorf("에스크로 해제 트랜잭션 실패: %v", err)
	}
	
	log.Printf("✅ 에스크로 해제 완료 - 사용자: %s, 주문: %s", userID, orderID)
	return nil
}

// getAvailableBalance는 사용자의 사용 가능한 잔액을 반환합니다.
func getAvailableBalance(userID string) (*ptypes.Account, error) {
	account, err := getUserAccount(userID)
	if err != nil {
		return nil, err
	}
	
	// 에스크로 계정이 초기화되지 않은 경우 초기화
	if account.EscrowAccount.FrozenPoliticianCoins == nil {
		account.EscrowAccount.FrozenPoliticianCoins = make(map[string]int64)
	}
	if account.EscrowAccount.ActiveOrders == nil {
		account.EscrowAccount.ActiveOrders = []string{}
	}
	
	return account, nil
}

// handleGetDepositAddress는 사용자의 테더코인 입금 주소를 반환합니다.
func handleGetDepositAddress(w http.ResponseWriter, r *http.Request) {
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
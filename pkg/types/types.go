package types

// TxData는 클라이언트가 전송하는 트랜잭션의 표준 구조입니다.
type TxData struct {
	TxID           string   `json:"tx_id,omitempty"`        // 트랜잭션 중복 방지용 고유 ID
	Action         string   `json:"action"` // "create_profile", "update_supporters", "propose_politician", "vote_on_proposal"
	UserID         string   `json:"user_id,omitempty"`
	Email          string   `json:"email,omitempty"`
	WalletAddress  string   `json:"wallet_address,omitempty"`  // Web3Auth에서 생성된 실제 지갑 주소
	PoliticianName string   `json:"politician_name,omitempty"`
	Region         string   `json:"region,omitempty"`
	Party          string   `json:"party,omitempty"`
	IntroUrl       string   `json:"intro_url,omitempty"`
	Politicians    []string `json:"politicians,omitempty"` // 지지하는 정치인 이름 목록
	ProposalID     string   `json:"proposal_id,omitempty"`
	Vote           bool     `json:"vote,omitempty"`
	Referrer       string   `json:"referrer,omitempty"`    // 추천인 지갑 주소
}

// ProfileInfoResponse는 사용자 프로필 조회 시 반환되는 데이터 구조입니다.
type ProfileInfoResponse struct {
	Email           string              `json:"email"`
	Nickname        string              `json:"nickname"`
	Wallet          string              `json:"wallet"`
	Country         string              `json:"country"`
	Gender          string              `json:"gender"`
	BirthYear       int                 `json:"birthYear"`
	Politisians     []string            `json:"politisians"`
	Balance         int64               `json:"balance"`               // 총 코인 잔액 (모든 정치인 코인의 합)
	ReferralCredits int                 `json:"referral_credits"`
	PoliticianCoins map[string]int64    `json:"politician_coins"`     // 정치인별 코인 보유량
	TotalCoins      int64               `json:"total_coins"`          // 총 코인 수 (편의용)
	TetherBalance   int64               `json:"tether_balance"`       // 테더코인 잔액
}

// ProfileSaveRequest는 프로필 저장 요청 시 받는 데이터 구조입니다.
type ProfileSaveRequest struct {
	Nickname    string   `json:"nickname"`
	Wallet      string   `json:"wallet"`
	Country     string   `json:"country"`
	Gender      string   `json:"gender"`
	BirthYear   int      `json:"birthYear"`
	Politisians []string `json:"politisians"`
	Referrer    string   `json:"referrer,omitempty"`
}

// Account는 사용자의 계정 정보를 나타냅니다.
type Account struct {
	Address           string              `json:"address"`
	Email             string              `json:"email,omitempty"`
	Wallet            string              `json:"wallet,omitempty"`  // PIN 기반으로 생성된 지갑 주소  
	Politicians       []string            `json:"politicians"`
	ReferralCredits   int                 `json:"referral_credits"`
	PoliticianCoins   map[string]int64    `json:"politician_coins"`   // "정치인명": 코인 보유량
	ReceivedCoins     map[string]bool     `json:"received_coins"`     // "정치인명": 코인 받았는지 여부
	InitialSelection  bool                `json:"initial_selection"`  // 초기 3명 선택 완료 여부
	TetherBalance     int64               `json:"tether_balance"`     // 테더코인 잔액 (USDT 단위)
	TetherWalletAddress string            `json:"tether_wallet_address"` // 테더코인 입금용 지갑 주소
	ActiveOrders      []TradeOrder        `json:"active_orders"`      // 활성 거래 주문들
	EscrowAccount     EscrowAccount       `json:"escrow_account"`     // 에스크로 계정
}

// Politician은 정치인의 정보를 나타냅니다.
type Politician struct {
	Name             string   `json:"name"`
	Region           string   `json:"region"`
	Party            string   `json:"party"`
	IntroUrl         string   `json:"intro_url,omitempty"`
	Supporters       []string `json:"supporters"`
	TotalCoinSupply  int64    `json:"total_coin_supply"`   // 총 발행량 (1,000만개)
	RemainingCoins   int64    `json:"remaining_coins"`     // 남은 코인 수량
	DistributedCoins int64    `json:"distributed_coins"`   // 이미 배포된 코인 수량
}

// Proposal은 새로운 정치인을 등록하기 위한 제안을 나타냅니다.
type Proposal struct {
	ID         string     `json:"id"`
	Politician Politician `json:"politician"`
	Proposer   string     `json:"proposer"`
	Votes      map[string]bool `json:"votes"`
	YesVotes   int        `json:"yes_votes"`
	NoVotes    int        `json:"no_votes"`
}

// ProposePolitisianRequest는 정치인 발의 API 요청을 위한 구조체입니다.
type ProposePolitisianRequest struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	Party    string `json:"party"`
	IntroUrl string `json:"introUrl,omitempty"`
}

// VoteRequest는 투표 API 요청을 위한 구조체입니다.
type VoteRequest struct {
	Vote bool `json:"vote"`
}

// User는 전통적 로그인 사용자 정보를 나타냅니다.
type User struct {
	ID           string `json:"id"`           // 고유 사용자 ID
	Email        string `json:"email"`        // 이메일 주소
	PasswordHash string `json:"password_hash"` // bcrypt 해시된 비밀번호
	Nickname     string `json:"nickname"`     // 닉네임
	PIN          string `json:"pin"`          // 지갑 PIN (해시됨)
	CreatedAt    int64  `json:"created_at"`   // 생성 시간
	IsActive     bool   `json:"is_active"`    // 활성 상태
}

// SignupRequest는 회원가입 요청 구조체입니다.
type SignupRequest struct {
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Nickname    string   `json:"nickname"`
	PIN         string   `json:"pin"`
	Politicians []string `json:"politicians"`
}

// LoginRequest는 로그인 요청 구조체입니다.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse는 로그인 응답 구조체입니다.
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	UserID  string `json:"user_id,omitempty"`
}

// TradeOrder는 거래 주문을 나타냅니다.
type TradeOrder struct {
	ID            string    `json:"id"`             // 주문 ID
	UserID        string    `json:"user_id"`        // 주문한 사용자 ID
	PoliticianID  string    `json:"politician_id"`  // 거래할 정치인 ID
	OrderType     string    `json:"order_type"`     // "buy" 또는 "sell"
	Quantity      int64     `json:"quantity"`       // 수량
	Price         int64     `json:"price"`          // 가격 (테더코인 단위)
	Status        string    `json:"status"`         // "active", "filled", "cancelled", "partial"
	FilledQuantity int64    `json:"filled_quantity"` // 체결된 수량
	EscrowAmount   int64    `json:"escrow_amount"`   // 에스크로 동결 금액
	CreatedAt     int64     `json:"created_at"`     // 생성 시간
	UpdatedAt     int64     `json:"updated_at"`     // 업데이트 시간
}

// EscrowAccount는 에스크로 계정을 나타냅니다.
type EscrowAccount struct {
	UserID              string            `json:"user_id"`              // 사용자 ID
	FrozenTetherBalance int64            `json:"frozen_tether_balance"` // 동결된 테더코인
	FrozenPoliticianCoins map[string]int64 `json:"frozen_politician_coins"` // 동결된 정치인 코인들
	ActiveOrders        []string         `json:"active_orders"`        // 활성 주문 ID 목록
}

// Trade는 체결된 거래를 나타냅니다.
type Trade struct {
	ID           string `json:"id"`            // 거래 ID
	BuyOrderID   string `json:"buy_order_id"`  // 매수 주문 ID
	SellOrderID  string `json:"sell_order_id"` // 매도 주문 ID
	BuyerID      string `json:"buyer_id"`      // 구매자 ID
	SellerID     string `json:"seller_id"`     // 판매자 ID
	PoliticianID string `json:"politician_id"` // 정치인 ID
	Quantity     int64  `json:"quantity"`      // 거래 수량
	Price        int64  `json:"price"`         // 거래 가격
	TotalAmount  int64  `json:"total_amount"`  // 총 거래 금액 (수량 × 가격)
	Timestamp    int64  `json:"timestamp"`     // 거래 시간
	Status       string `json:"status"`        // "completed", "processing"
}

// OrderBook은 특정 정치인의 오더북을 나타냅니다.
type OrderBook struct {
	PoliticianID  string       `json:"politician_id"`
	BuyOrders     []TradeOrder `json:"buy_orders"`    // 매수 주문들 (가격 높은 순)
	SellOrders    []TradeOrder `json:"sell_orders"`   // 매도 주문들 (가격 낮은 순)
	LastPrice     int64        `json:"last_price"`    // 최근 체결가
	Volume24h     int64        `json:"volume_24h"`    // 24시간 거래량
}

// TradeRequest는 거래 주문 요청을 나타냅니다.
type TradeRequest struct {
	PoliticianID  string `json:"politician_id"`  // 거래할 정치인 ID
	OrderType     string `json:"order_type"`     // "buy" 또는 "sell"
	Quantity      int64  `json:"quantity"`       // 수량
	Price         int64  `json:"price"`          // 가격 (테더코인 단위)
	PIN           string `json:"pin"`            // 거래 승인용 PIN
}

// DepositRequest는 테더코인 입금 요청을 나타냅니다.
type DepositRequest struct {
	Amount     int64  `json:"amount"`      // 입금 금액 (USDT)
	TxHash     string `json:"tx_hash"`     // 블록체인 트랜잭션 해시
	FromAddress string `json:"from_address"` // 송금한 주소
	PIN        string `json:"pin"`         // 입금 승인용 PIN
}

// WithdrawRequest는 테더코인 출금 요청을 나타냅니다.
type WithdrawRequest struct {
	Amount    int64  `json:"amount"`     // 출금 금액 (USDT)
	ToAddress string `json:"to_address"` // 받을 주소
	PIN       string `json:"pin"`        // 출금 승인용 PIN
}

// PoliticianPrice는 정치인 코인의 가격 정보를 나타냅니다.
type PoliticianPrice struct {
	PoliticianID   string `json:"politician_id"`
	Name           string `json:"name"`
	CurrentPrice   int64  `json:"current_price"`   // 현재 가격
	Change24h      int64  `json:"change_24h"`      // 24시간 변동가
	ChangePercent  float64 `json:"change_percent"` // 24시간 변동률
	Volume24h      int64  `json:"volume_24h"`      // 24시간 거래량
	Rank           int    `json:"rank"`            // 가격 순위
}

// GenesisState는 블록체인의 초기 상태를 정의합니다.
type GenesisState struct {
	Accounts       map[string]*Account       `json:"accounts"`
	Politicians    map[string]*Politician    `json:"politicians"`
	Users          map[string]*User          `json:"users"`          // 사용자 정보 추가
	Orders         []TradeOrder              `json:"orders"`         // 거래 주문들
	EscrowAccounts map[string]*EscrowAccount `json:"escrow_accounts"` // 에스크로 계정들
	Trades         []Trade                   `json:"trades"`         // 체결된 거래 기록들
} 
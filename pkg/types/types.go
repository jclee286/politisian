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

// GenesisState는 블록체인의 초기 상태를 정의합니다.
type GenesisState struct {
	Accounts    map[string]*Account    `json:"accounts"`
	Politicians map[string]*Politician `json:"politicians"`
} 
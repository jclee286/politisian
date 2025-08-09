package types

// TxData는 클라이언트가 전송하는 트랜잭션의 표준 구조입니다.
type TxData struct {
	Action         string   `json:"action"` // "create_profile", "update_supporters", "propose_politician", "vote_on_proposal"
	UserID         string   `json:"user_id,omitempty"`
	Email          string   `json:"email,omitempty"`
	PoliticianName string   `json:"politician_name,omitempty"`
	Region         string   `json:"region,omitempty"`
	Party          string   `json:"party,omitempty"`
	Politicians    []string `json:"politicians,omitempty"` // 지지하는 정치인 이름 목록
	ProposalID     string   `json:"proposal_id,omitempty"`
	Vote           bool     `json:"vote,omitempty"`
}

// ProfileInfoResponse는 사용자 프로필 조회 시 반환되는 데이터 구조입니다.
type ProfileInfoResponse struct {
	Email           string   `json:"email"`
	Nickname        string   `json:"nickname"`
	Wallet          string   `json:"wallet"`
	Country         string   `json:"country"`
	Gender          string   `json:"gender"`
	BirthYear       int      `json:"birthYear"`
	Politisians     []string `json:"politisians"`
	Balance         int64    `json:"balance"`
	ReferralCredits int      `json:"referral_credits"`
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
	Address     string   `json:"address"`
	Email       string   `json:"email,omitempty"`
	Politicians []string `json:"politicians"`
}

// Politician은 정치인의 정보를 나타냅니다.
type Politician struct {
	Name         string   `json:"name"`
	Region       string   `json:"region"`
	Party        string   `json:"party"`
	Supporters   []string `json:"supporters"`
	TokensMinted int64    `json:"tokens_minted"`
	MaxTokens    int64    `json:"max_tokens"`
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
	Name   string `json:"name"`
	Region string `json:"region"`
	Party  string `json:"party"`
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
package types

// TxData는 블록체인으로 전송될 트랜잭션의 데이터 구조입니다.
type TxData struct {
	Action      string   `json:"action"` // "create_profile", "claim_reward", "propose_politician", "vote_on_proposal"
	Email       string   `json:"email,omitempty"`
	Nickname    string   `json:"nickname,omitempty"`
	Wallet      string   `json:"wallet"`
	Country     string   `json:"country,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	BirthYear   int      `json:"birthYear,omitempty"`
	Politicians []string `json:"politicians,omitempty"`
	Referrer    string   `json:"referrer,omitempty"`

	// 정치인 발의 관련
	ProposalID   string `json:"proposal_id,omitempty"`
	PoliticianName string `json:"politician_name,omitempty"`
	Region       string `json:"region,omitempty"`
	Party        string `json:"party,omitempty"`

	// 투표 관련
	Vote bool `json:"vote,omitempty"` // true for '찬성', false for '반대'
}

// ProfileInfoResponse는 사용자 프로필 조회 시 반환되는 데이터 구조입니다.
type ProfileInfoResponse struct {
	Email           string   `json:"email"`
	Nickname        string   `json:"nickname"`
	Wallet          string   `json:"wallet"`
	Country         string   `json:"country"`
	Gender          string   `json:"gender"`
	BirthYear       int      `json:"birthYear"`
	Politicians     []string `json:"politicians"`
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
	Politicians []string `json:"politicians"`
	Referrer    string   `json:"referrer,omitempty"`
}

// Account는 블록체인 상에 저장되는 사용자의 모든 프로필 정보를 나타냅니다.
type Account struct {
	Email           string   `json:"email"`
	Nickname        string   `json:"nickname"`
	Wallet          string   `json:"wallet"`
	Country         string   `json:"country"`
	Gender          string   `json:"gender"`
	BirthYear       int      `json:"birthYear"`
	Politicians     []string `json:"politicians"`
	Balance         int64    `json:"balance"`
	Referrer        string   `json:"referrer,omitempty"`
	ReferralCredits int      `json:"referral_credits"`
}

// Politician은 블록체인 상에 저장되는 정치인의 데이터 구조입니다.
type Politician struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	Party        string `json:"party"`
	TokensMinted int64  `json:"tokens_minted"`
	MaxTokens    int64  `json:"max_tokens"`
}

// Proposal은 정치인 등록 제안을 나타내는 데이터 구조입니다.
type Proposal struct {
	ID          string            `json:"id"`
	Politician  Politician        `json:"politician"`
	Proposer    string            `json:"proposer"` // 제안자의 이메일
	Votes       map[string]bool   `json:"votes"`    // 투표자(이메일) -> 투표 내용 (true: 찬성, false: 반대)
	YesVotes    int               `json:"yes_votes"`
	NoVotes     int               `json:"no_votes"`
}

// ProposePoliticianRequest는 정치인 발의 API 요청을 위한 구조체입니다.
type ProposePoliticianRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	Party  string `json:"party"`
}

// VoteRequest는 투표 API 요청을 위한 구조체입니다.
type VoteRequest struct {
	Vote bool `json:"vote"`
} 
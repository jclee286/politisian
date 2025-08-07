package types

// TxData는 블록체인으로 전송될 트랜잭션의 데이터 구조입니다.
type TxData struct {
	Action      string   `json:"action"` // "create_profile" 또는 "claim_reward"
	Email       string   `json:"email,omitempty"`
	Nickname    string   `json:"nickname,omitempty"`
	Wallet      string   `json:"wallet"`
	Country     string   `json:"country,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	BirthYear   int      `json:"birthYear,omitempty"`
	Politicians []string `json:"politicians,omitempty"`
	Referrer    string   `json:"referrer,omitempty"`
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
	TokensMinted int64  `json:"tokens_minted"`
	MaxTokens    int64  `json:"max_tokens"`
} 
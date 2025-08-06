package server

// TxData는 블록체인으로 전송될 트랜잭션의 데이터 구조입니다.
type TxData struct {
	Email       string   `json:"email"`
	Nickname    string   `json:"nickname"`
	Wallet      string   `json:"wallet"`
	Country     string   `json:"country"`
	Gender      string   `json:"gender"`
	BirthYear   int      `json:"birthYear"`
	Politicians []string `json:"politicians"`
}

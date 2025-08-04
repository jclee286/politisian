package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleOauth2 "google.golang.org/api/oauth2/v2"
)

var (
	googleOauthConfig *oauth2.Config
	// TODO: 이 값들을 실제 운영 환경에서는 환경 변수나 다른 보안 저장소에서 불러와야 합니다.
	oauthClientID     = "152573583059-2k51btfpnqb31potv830g676nag3flps.apps.googleusercontent.com"
	oauthClientSecret = "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"
)

func main() {
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/auth/google/login", handleGoogleLogin)
	apiMux.HandleFunc("/auth/google/callback", handleGoogleCallback)

	http.Handle("/api/", http.StripPrefix("/api", apiMux))

	// 프론트엔드 파일을 서비스하기 위한 파일 서버 설정
	// "./politician/frontend/" 디렉토리의 파일을 웹 루트("/")에서 제공
	fs := http.FileServer(http.Dir("/home/jclee/politician/politician/frontend/"))
	http.Handle("/", fs)

	fmt.Println("서버 시작. http://localhost:8080 에서 수신 대기 중...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("서버 시작 실패: %v", err)
	}
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// CSRF 공격 방지를 위한 state 토큰 생성
	state, err := generateOauthState()
	if err != nil {
		http.Error(w, "State 생성 실패", http.StatusInternalServerError)
		return
	}
	// 생성된 state를 쿠키에 저장
	http.SetCookie(w, &http.Cookie{
		Name:    "oauthstate",
		Value:   state,
		Expires: time.Now().Add(10 * time.Minute),
	})

	// 사용자를 Google 인증 페이지로 리디렉션
	url := googleOauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// 쿠키에서 state 값 가져오기
	oauthState, _ := r.Cookie("oauthstate")

	// CSRF 공격 확인: 요청의 state와 쿠키의 state가 일치하는지 확인
	if r.FormValue("state") != oauthState.Value {
		log.Println("Invalid oauth google state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// state 확인 후 사용자 정보 가져오기
	data, err := getUserDataFromGoogle(r.FormValue("code"))
	if err != nil {
		log.Println(err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// 사용자 이메일을 기반으로 회원가입 트랜잭션 생성
	// 실제 앱에서는 여기서 사용자에게 3명의 정치인을 선택하도록 유도해야 합니다.
	// 지금은 테스트를 위해 임의의 정치인 ID를 사용합니다.
	txPayload := map[string]interface{}{
		"action":      "signup",
		"user":        data.Email,
		"politicians": []string{"p1", "p2", "p3"},
	}

	// 트랜잭션을 블록체인에 전송
	if err := sendTxToBlockchain(txPayload); err != nil {
		log.Printf("블록체인 트랜잭션 전송 실패: %v", err)
		http.Error(w, "회원가입 처리 중 오류가 발생했습니다.", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "로그인 및 블록체인 회원가입 성공!\n사용자 정보:\n%s\n", data)
}

func generateOauthState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func getUserDataFromGoogle(code string) (*googleOauth2.Userinfo, error) {
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %s", err.Error())
	}

	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}

	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %s", err.Error())
	}

	var userInfo googleOauth2.Userinfo
	if err := json.Unmarshal(contents, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %s", err.Error())
	}

	return &userInfo, nil
}

// 블록체인 노드에 트랜잭션을 전송하는 헬퍼 함수
func sendTxToBlockchain(payload map[string]interface{}) error {
	// 페이로드를 JSON 문자열로 변환
	txBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON 변환 실패: %w", err)
	}

	// CometBFT RPC 엔드포인트 URL 생성
	txHex := fmt.Sprintf("0x%x", txBytes)
	rpcURL := fmt.Sprintf("http://localhost:26657/broadcast_tx_sync?tx=%s", url.QueryEscape(txHex))

	// CometBFT 노드에 트랜잭션 전송
	resp, err := http.Get(rpcURL)
	if err != nil {
		return fmt.Errorf("RPC 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	// 응답 확인
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("RPC 응답 읽기 실패: %w", err)
	}

	// 응답에 에러가 있는지 확인 (code가 0이 아니면 에러)
	var rpcResponse struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Code uint32 `json:"code"`
			Log  string `json:"log"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return fmt.Errorf("RPC 응답 JSON 파싱 실패: %w, 응답: %s", err, string(body))
	}

	if rpcResponse.Error != nil {
		return fmt.Errorf("RPC 에러: %s (%s)", rpcResponse.Error.Message, rpcResponse.Error.Data)
	}

	if rpcResponse.Result.Code != 0 {
		return fmt.Errorf("ABCI 에러: %s (코드: %d)", rpcResponse.Result.Log, rpcResponse.Result.Code)
	}

	log.Printf("블록체인 트랜잭션 성공: %s", txHex)
	return nil
} 
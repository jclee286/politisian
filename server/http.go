package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/cometbft/cometbft/abci/types" // 상세한 오류 처리를 위해 추가
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/rpc/client/local"
	"golang.org/x/oauth2"
	googleOauth2 "google.golang.org/api/oauth2/v2"
)

// TxData는 이제 블록체인으로 전송될 모든 프로필 정보를 포함합니다.
type TxData struct {
	Email       string   `json:"email"`
	Nickname    string   `json:"nickname"`
	Wallet      string   `json:"wallet"`
	Country     string   `json:"country"`
	Gender      string   `json:"gender"`
	BirthYear   int      `json:"birthYear"`
	Politicians []string `json:"politicians"`
}

// UserData는 프로필 설정 중인 신규 사용자의 정보를 임시로 저장합니다.
type UserData struct {
	Email  string
	Wallet string
}

var (
	googleOauthConfig *oauth2.Config
	blockchainClient  *local.Local
	// 동시성 문제 방지를 위해 sync.Map 사용
	tempUserStore = &sync.Map{}
)

func StartHTTPServer(client *local.Local, listenAddr, homeDir string) {
	blockchainClient = client
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		ClientID:     "152573583059-2k51btfpnqb31potv830g676nag3flps.apps.googleusercontent.com", // 사용자님 ID로 교체
		ClientSecret: "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",                                     // 사용자님 비밀번호로 교체
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	fs := http.FileServer(http.Dir(homeDir + "/frontend/"))
	http.Handle("/", fs)
	http.HandleFunc("/api/auth/google/login", handleGoogleLogin)
	http.HandleFunc("/api/auth/google/callback", handleGoogleCallback)
	// 프로필 설정을 위한 신규 핸들러 추가
	http.HandleFunc("/api/me/profile-info", handleGetProfileInfo)
	http.HandleFunc("/api/profile/save", handleProfileSave)

	log.Printf("HTTP 서버 시작: %s (프론트엔드 경로: %s)", listenAddr, homeDir+"/frontend/")
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("HTTP 서버 시작 실패: %v", err)
	}
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	rand.Read(b)
	oauthStateString := base64.URLEncoding.EncodeToString(b)

	cookie := http.Cookie{
		Name:    "oauthstate",
		Value:   oauthStateString,
		Expires: time.Now().Add(10 * time.Minute),
		Path:    "/", // 쿠키가 모든 경로에서 유효하도록 설정합니다.
	}
	http.SetCookie(w, &cookie)

	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	log.Println("Google 콜백 처리 시작...")
	state, err := r.Cookie("oauthstate")
	if err != nil {
		log.Printf("State 쿠키 읽기 오류: %v", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if r.FormValue("state") != state.Value {
		log.Printf("유효하지 않은 oauth state: 수신=%s, 쿠키=%s", r.FormValue("state"), state.Value)
		http.Error(w, "유효하지 않은 oauth state", http.StatusBadRequest)
		return
	}

	data, err := GetUserDataFromGoogle(r.FormValue("code"))
	if err != nil {
		log.Printf("Google로부터 사용자 정보 얻기 실패: %v", err)
		http.Error(w, "Google로부터 사용자 정보 얻기 실패", http.StatusInternalServerError)
		return
	}

	log.Printf("사용자 정보 수신: %s", data.Email)
	exists, err := checkUserExists(data.Email)
	if err != nil {
		log.Printf("사용자 확인 중 오류: %v", err)
		http.Error(w, "사용자 확인 중 오류가 발생했습니다.", http.StatusInternalServerError)
		return
	}

	if exists {
		log.Printf("기존 사용자 로그인: %s", data.Email)
		// TODO: 기존 사용자를 위한 메인 페이지를 만들어야 합니다.
		// 지금은 임시로 프로필 페이지로 보내지만, 개선이 필요합니다.
		http.Redirect(w, r, "/profile.html", http.StatusTemporaryRedirect)
	} else {
		log.Printf("신규 사용자 감지: %s", data.Email)

		// 1. 새 지갑 주소 생성
		privKey := ed25519.GenPrivKey()
		walletAddress := privKey.PubKey().Address().String()
		log.Printf("새 지갑 주소 생성: %s", walletAddress)

		// 2. oauth state를 키로 사용하여 사용자 정보를 임시 저장
		tempUserStore.Store(state.Value, UserData{
			Email:  data.Email,
			Wallet: walletAddress,
		})
		log.Printf("임시 저장소에 사용자 정보 저장 (key: %s)", state.Value)

		// 3. 프로필 설정 페이지로 리디렉션
		http.Redirect(w, r, "/profile.html", http.StatusTemporaryRedirect)
	}
}

// handleGetProfileInfo는 페이지 로드 시 프론트엔드에 지갑주소 등을 전달합니다.
func handleGetProfileInfo(w http.ResponseWriter, r *http.Request) {
	log.Println("프로필 정보 요청 수신")
	state, err := r.Cookie("oauthstate")
	if err != nil {
		http.Error(w, "인증 상태를 찾을 수 없습니다. 다시 로그인해주세요.", http.StatusUnauthorized)
		return
	}

	userData, ok := tempUserStore.Load(state.Value)
	if !ok {
		http.Error(w, "사용자 정보를 찾을 수 없습니다. 다시 로그인해주세요.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userData)
	log.Printf("프로필 정보 전송: %v", userData)
}

// handleProfileSave는 프론트엔드에서 보낸 프로필 정보를 받아 블록체인에 기록합니다.
func handleProfileSave(w http.ResponseWriter, r *http.Request) {
	log.Println("프로필 저장 요청 수신")
	if r.Method != http.MethodPost {
		http.Error(w, "POST 요청만 허용됩니다.", http.StatusMethodNotAllowed)
		return
	}

	state, err := r.Cookie("oauthstate")
	if err != nil {
		http.Error(w, "인증 상태를 찾을 수 없습니다. 다시 로그인해주세요.", http.StatusUnauthorized)
		return
	}

	userData, ok := tempUserStore.Load(state.Value)
	if !ok {
		http.Error(w, "만료된 요청입니다. 다시 로그인해주세요.", http.StatusNotFound)
		return
	}

	var profileData struct {
		Country     string   `json:"country"`
		Gender      string   `json:"gender"`
		BirthYear   int      `json:"birthYear"`
		Politicians []string `json:"politicians"`
	}
	if err := json.NewDecoder(r.Body).Decode(&profileData); err != nil {
		http.Error(w, "잘못된 요청 데이터입니다.", http.StatusBadRequest)
		return
	}

	// 임시 정보와 폼 데이터를 합쳐 완전한 트랜잭션 데이터 생성
	txData := TxData{
		Email:       userData.(UserData).Email,
		Wallet:      userData.(UserData).Wallet,
		Nickname:    "NewUser", // 닉네임은 추후 구글 프로필 등에서 가져올 수 있음
		Country:     profileData.Country,
		Gender:      profileData.Gender,
		BirthYear:   profileData.BirthYear,
		Politicians: profileData.Politicians,
	}

	txBytes, err := json.Marshal(txData)
	if err != nil {
		http.Error(w, "트랜잭션 생성 실패", http.StatusInternalServerError)
		return
	}

	// 트랜잭션이 블록에 포함될 때까지 기다리는 Commit 사용
	res, err := blockchainClient.BroadcastTxCommit(context.Background(), txBytes)
	if err != nil {
		log.Printf("BroadcastTxCommit RPC 실패: %v", err)
		http.Error(w, "블록체인에 프로필 저장 실패 (RPC 오류)", http.StatusInternalServerError)
		return
	}

	// CheckTx 결과를 확인하여 트랜잭션 유효성 검사 실패를 확인합니다.
	if res.CheckTx.Code != types.CodeTypeOK {
		log.Printf("CheckTx 실패: code=%d, log=%s", res.CheckTx.Code, res.CheckTx.Log)
		http.Error(w, fmt.Sprintf("블록체인 트랜잭션 확인 실패: %s", res.CheckTx.Log), http.StatusInternalServerError)
		return
	}

	// FinalizeBlock (DeliverTx) 결과를 확인하여 트랜잭션 실행 실패를 확인합니다.
	// v0.38부터는 DeliverTx가 TxResult로 이름이 변경되었습니다.
	if res.TxResult.Code != types.CodeTypeOK {
		log.Printf("TxResult(FinalizeBlock) 실패: code=%d, log=%s", res.TxResult.Code, res.TxResult.Log)
		http.Error(w, fmt.Sprintf("블록체인 트랜잭션 실행 실패: %s", res.TxResult.Log), http.StatusInternalServerError)
		return
	}

	// 사용 완료된 임시 정보 삭제
	tempUserStore.Delete(state.Value)
	log.Printf("프로필 저장 성공 및 임시 정보 삭제: %s", userData.(UserData).Email)

	// TODO: 성공 후 보낼 실제 메인 페이지가 필요함
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("프로필이 성공적으로 저장되었습니다!"))
}

func GetUserDataFromGoogle(code string) (*googleOauth2.Userinfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := googleOauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	var userInfo googleOauth2.Userinfo
	err = json.Unmarshal(contents, &userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling user info: %s", err.Error())
	}

	return &userInfo, nil
}

func checkUserExists(email string) (bool, error) {
	log.Printf("사용자 존재 여부 확인: %s", email)
	queryData := []byte(email)

	// 특정 경로로 쿼리하여 앱에서 분기 처리하도록 함
	res, err := blockchainClient.ABCIQuery(context.Background(), "/account/exists", queryData)
	if err != nil {
		log.Printf("ABCIQuery 실패: %v", err)
		return false, fmt.Errorf("ABCIQuery failed: %w", err)
	}

	if res.Response.Code == 0 {
		log.Printf("사용자 '%s'가 존재합니다.", email)
		return true, nil
	}

	log.Printf("사용자 '%s'가 존재하지 않습니다. (응답 코드: %d)", email, res.Response.Code)
	return false, nil
} 
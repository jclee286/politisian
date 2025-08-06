package server

import (
	"log"
	"net/http"
	"os"

	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/rpc/client/local"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// 전역 변수들은 이제 각 파일에서 필요한 만큼만 선언되거나, 여기서처럼 공유됩니다.
var (
	googleOauthConfig *oauth2.Config
	blockchainClient  *local.Local
)

// TxData 구조체는 핸들러 파일에서만 사용되므로 여기서는 삭제합니다.

// StartServer는 HTTP 서버를 시작하고 모든 라우팅을 설정합니다.
func StartServer(node *node.Node) {
	// Google OAuth 설정
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}

	if googleOauthConfig.ClientID == "" || googleOauthConfig.ClientSecret == "" {
		log.Println("경고: GOOGLE_CLIENT_ID 또는 GOOGLE_CLIENT_SECRET 환경 변수가 설정되지 않았습니다.")
	}

	log.Println("서버 시작: http://localhost:8080")
	blockchainClient = local.New(node)

	// 정적 파일 서버 설정
	fs := http.FileServer(http.Dir("frontend"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	
	// 기본 페이지 라우팅
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "frontend/login.html")
			return
		}
		http.ServeFile(w, r, "frontend"+r.URL.Path)
	})

	// API 라우팅 설정 (분리된 함수들 호출)
	http.HandleFunc("/api/auth/google", handleGoogleLogin)
	http.HandleFunc("/api/auth/google/callback", handleGoogleCallback)
	http.HandleFunc("/api/me/profile-info", authMiddleware(handleGetProfileInfo))
	http.HandleFunc("/api/profile/save", authMiddleware(handleProfileSave))
	http.HandleFunc("/api/politicians", authMiddleware(handleGetPoliticians))
	http.HandleFunc("/api/me/dashboard", authMiddleware(handleDashboard))

	// HTTP 서버 시작
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("HTTP 서버 시작 실패: %v", err)
	}
}

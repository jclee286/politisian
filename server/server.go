package server

import (
	"log"
	"net/http"
	"os"

	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/rpc/client/local"
)

var blockchainClient *local.Local

func StartServer(node *node.Node) {
	// Google OAuth 클라이언트 정보 설정
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if googleClientID == "" || googleClientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set")
	}
	InitOauth(googleClientID, googleClientSecret)

	blockchainClient = local.New(node)

	// 인증이 필요 없는 라우트
	http.HandleFunc("/api/auth/google", handleGoogleLogin)
	http.HandleFunc("/api/auth/google/callback", handleGoogleCallback)

	// 인증이 필요한 API 라우트
	http.Handle("/api/me/profile-info", authMiddleware(http.HandlerFunc(handleGetProfileInfo)))
	http.Handle("/api/profile/save", authMiddleware(http.HandlerFunc(handleProfileSave)))
	http.Handle("/api/politicians", authMiddleware(http.HandlerFunc(handleGetPoliticians)))
	http.Handle("/api/me/dashboard", authMiddleware(http.HandlerFunc(handleDashboard)))

	// Frontend 파일 서빙
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	log.Println("HTTP server listening on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

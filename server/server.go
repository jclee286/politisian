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
		log.Println("WARNING: GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET not set. Authentication may not work.")
		// Continue without fatal
	} else {
		InitOauth(googleClientID, googleClientSecret)
	}

	blockchainClient = local.New(node)

	// 인증이 필요 없는 라우트
	http.HandleFunc("/api/auth/google", handleGoogleLogin)
	http.HandleFunc("/api/auth/google/callback", handleGoogleCallback)

	// 인증이 필요한 API 라우트
	http.Handle("/api/me/profile-info", authMiddleware(http.HandlerFunc(handleGetProfileInfo)))
	http.Handle("/api/profile/save", authMiddleware(http.HandlerFunc(handleProfileSave)))
	http.Handle("/api/politicians", authMiddleware(http.HandlerFunc(handleGetPoliticians)))
	http.Handle("/api/me/dashboard", authMiddleware(http.HandlerFunc(handleDashboard)))
	http.Handle("/api/rewards/claim", authMiddleware(http.HandlerFunc(handleClaimReward)))

	// Frontend 파일 서빙 with strict authentication for root and dashboard
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Allow login and auth routes without authentication
		if r.URL.Path == "/login.html" || r.URL.Path == "/api/auth/google" || r.URL.Path == "/api/auth/google/callback" {
			fs := http.FileServer(http.Dir("./frontend"))
			fs.ServeHTTP(w, r)
			return
		}

		// Check session token
		sessionCookie, err := r.Cookie("session_token")
		if err != nil || sessionCookie.Value == "" || sessionStore[sessionCookie.Value] == "" {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}

		// If authenticated, serve the file
		fs := http.FileServer(http.Dir("./frontend"))
		fs.ServeHTTP(w, r)
	})

	// Explicitly protect /index.html with the same logic
	http.HandleFunc("/index.html", func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil || sessionCookie.Value == "" || sessionStore[sessionCookie.Value] == "" {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}
		fs := http.FileServer(http.Dir("./frontend"))
		fs.ServeHTTP(w, r)
	})

	log.Println("HTTP server listening on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

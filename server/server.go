package server

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/rpc/client/local"
)

var blockchainClient *local.Local

func StartServer(node *node.Node) {
	blockchainClient = local.New(node)

	// --- 새로운 지갑 인증 라우트 ---
	http.HandleFunc("/api/auth/wallet/login", handleWalletLogin)

	// 인증이 필요 없는 라우트
	http.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./frontend/login.html")
	})

	// 인증이 필요한 API 라우트
	http.Handle("/api/me/profile-info", authMiddleware(http.HandlerFunc(handleGetProfileInfo)))
	http.Handle("/api/profile/save", authMiddleware(http.HandlerFunc(handleProfileSave)))
	http.Handle("/api/politicians", authMiddleware(http.HandlerFunc(handleGetPoliticians)))
	http.Handle("/api/me/dashboard", authMiddleware(http.HandlerFunc(handleDashboard)))
	http.Handle("/api/rewards/claim", authMiddleware(http.HandlerFunc(handleClaimReward)))

	// 거버넌스 관련 API 라우트 (인증 필요)
	http.Handle("/api/politicians/propose", authMiddleware(http.HandlerFunc(handleProposePolitician)))
	http.Handle("/api/proposals", authMiddleware(http.HandlerFunc(handleGetProposals)))
	http.HandleFunc("/api/proposals/", func(w http.ResponseWriter, r *http.Request) {
		// /api/proposals/{id}/vote 형태의 경로를 처리
		if strings.HasSuffix(r.URL.Path, "/vote") {
			authMiddleware(http.HandlerFunc(handleVoteOnProposal)).ServeHTTP(w, r)
			return
		}
		// 다른 /api/proposals/ 경로는 404 처리
		http.NotFound(w, r)
	})

	// 그 외 모든 정적 파일 요청은 인증 미들웨어를 거침
	http.Handle("/", authFileServerMiddleware())

	log.Println("HTTP server listening on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// authFileServerMiddleware는 정적 파일 요청에 대한 인증을 처리하는 미들웨어입니다.
func authFileServerMiddleware() http.Handler {
	fs := http.FileServer(http.Dir("./frontend"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		// 쿠키가 없거나 세션이 유효하지 않으면 로그인 페이지로 리디렉션
		if err != nil || sessionCookie.Value == "" {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}
		if _, exists := sessionStore.Get(sessionCookie.Value); !exists {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}

		// 요청된 경로에 해당하는 파일이 없으면 index.html을 서빙 (SPA 지원)
		if _, err := os.Stat("./frontend" + r.URL.Path); os.IsNotExist(err) {
			http.ServeFile(w, r, "./frontend/index.html")
			return
		}

		// 인증되었다면 요청된 파일 서빙
		fs.ServeHTTP(w, r)
	})
}

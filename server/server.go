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
	blockchainClient = local.New(node)

	// API 엔드포인트 라우팅을 먼저 설정합니다.
	// 이렇게 하면 /api/ 요청이 파일 서버에 의해 가로채지지 않습니다.
	http.Handle("/api/auth/wallet/login", http.HandlerFunc(handleWalletLogin))
	http.Handle("/api/user/profile", authMiddleware(http.HandlerFunc(handleUserProfile)))
	http.Handle("/api/profile/save", authMiddleware(http.HandlerFunc(handleProfileSave)))
	http.Handle("/api/politisian/list", authMiddleware(http.HandlerFunc(handleGetPolitisians)))
	http.Handle("/api/politisian/propose", authMiddleware(http.HandlerFunc(handleProposePolitician)))

	// 정적 파일을 제공하는 파일 서버를 설정합니다. 이 핸들러는 API 핸들러 뒤에 위치해야 합니다.
	fs := http.FileServer(http.Dir("./frontend/"))
	http.Handle("/", fs)

	// 서버 시작
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("HTTP server listening on :%s", port)
	err := http.ListenAndServe(":"+port, nil)
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

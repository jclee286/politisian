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

	// 클라이언트 측 자산을 제공하기 위해 파일 서버를 설정합니다.
	// http.Dir("./frontend/")는 ./frontend/ 디렉토리의 파일을 사용하도록 지정합니다.
	// http.StripPrefix를 사용하지 않고, 모든 요청을 파일 서버로 보냅니다.
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	// API 엔드포인트 라우팅 설정
	// API 핸들러가 파일 서버 핸들러보다 먼저 등록되어야 /api/ 경로가 올바르게 처리됩니다.
	http.HandleFunc("/api/auth/wallet/login", handleWalletLogin)
	http.Handle("/api/user/profile", authMiddleware(http.HandlerFunc(handleUserProfile)))
	http.Handle("/api/profile/save", authMiddleware(http.HandlerFunc(handleProfileSave)))
	
	// 정치인 관련 API
	http.Handle("/api/politisian/list", authMiddleware(http.HandlerFunc(handleGetPolitisians)))
	http.Handle("/api/politisian/propose", authMiddleware(http.HandlerFunc(handleProposePolitician)))


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

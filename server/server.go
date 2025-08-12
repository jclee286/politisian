package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/rpc/client/local"
)

var blockchainClient *local.Local

func StartServer(node *node.Node) {
	blockchainClient = local.New(node)

	mux := http.NewServeMux()

	// 1. API 핸들러들을 먼저 명확하게 등록합니다.
	mux.HandleFunc("/api/auth/wallet/login", handleWalletLogin)
	mux.Handle("/api/user/profile", authMiddleware(http.HandlerFunc(handleUserProfile)))
	mux.Handle("/api/profile/save", authMiddleware(http.HandlerFunc(handleProfileSave)))
	mux.Handle("/api/politisian/list", authMiddleware(http.HandlerFunc(handleGetPolitisians)))
	mux.Handle("/api/politisian/propose", authMiddleware(http.HandlerFunc(handleProposePolitician)))
	// 나중에 추가될 API 핸들러들...

	// 2. 정적 파일 핸들러 (CSS, JS 등)를 등록합니다. 이 요청들은 인증을 거치지 않습니다.
	// ./frontend/js/ 디렉토리를 /js/ URL 경로에 매핑합니다.
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./frontend/js"))))

	// 3. 그 외 모든 페이지 요청을 처리할 핸들러를 등록합니다. (가장 마지막에 위치)
	fs := http.FileServer(http.Dir("./frontend"))
	mux.HandleFunc("/", rootFileHandler(fs))

	// 서버 시작
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("HTTP server listening on :%s", port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// rootFileHandler는 서버 사이드에서 인증을 확인하고 파일을 서빙하는 똑똑한 핸들러입니다.
func rootFileHandler(fs http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		// API 요청은 이미 위에서 처리되었으므로 여기로 오지 않습니다.
		
		// login.html, favicon.ico 등 인증이 필요 없는 파일들은 그냥 보여줍니다.
		if r.URL.Path == "/login.html" || r.URL.Path == "/favicon.ico" {
			fs.ServeHTTP(w, r)
			return
		}

		// 그 외의 모든 페이지 요청(예: /, /index.html, /profile.html)은 인증을 확인합니다.
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			// 쿠키가 없으면 로그인 페이지로 리다이렉트합니다.
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		if _, exists := sessionStore.Get(sessionCookie.Value); !exists {
			// 유효하지 않은 세션이면 쿠키를 삭제하고 로그인 페이지로 리다이렉트합니다.
			http.SetCookie(w, &http.Cookie{Name: "session_token", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		// 인증된 사용자입니다. 요청한 파일을 보여줍니다.
		// 단, 경로가 / 이면 /index.html을 보여줍니다.
		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}

		// 요청된 파일이 실제로 존재하는지 확인하고 없으면 index.html을 보여줍니다(SPA 방식).
		if _, err := os.Stat("./frontend" + r.URL.Path); os.IsNotExist(err) {
			http.ServeFile(w, r, "./frontend/index.html")
			return
		}
		
		fs.ServeHTTP(w, r)
	}
}

// authMiddleware는 API 요청에 대한 인증을 처리합니다. (기존 코드와 거의 동일)
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("AuthMiddleware: Checking %s", r.URL.Path)
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			log.Printf("AuthMiddleware: Failed. No session token. %s", r.URL.Path)
			http.Error(w, "Unauthorized: No session token", http.StatusUnauthorized)
			return
		}

		sessionToken := sessionCookie.Value
		userID, exists := sessionStore.Get(sessionToken)
		if !exists {
			log.Printf("AuthMiddleware: Failed. Invalid session token. %s", r.URL.Path)
			http.Error(w, "Unauthorized: Invalid session token", http.StatusUnauthorized)
			return
		}

		log.Printf("AuthMiddleware: Success. User %s authorized for %s", userID, r.URL.Path)
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

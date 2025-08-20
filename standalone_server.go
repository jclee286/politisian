package main

import (
	"log"
	"net/http"
	"os"
	
	"github.com/jclee286/politisian/server"
)

// 임시 standalone 서버 - 블록체인 노드 없이 실행
func main() {
	log.Println("🚀 임시 standalone 서버 시작")
	
	// 간단한 HTTP 서버 시작
	mux := http.NewServeMux()
	
	// CORS 미들웨어
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
	
	// 인증 API만 등록
	mux.Handle("/api/auth/signup", corsMiddleware(http.HandlerFunc(server.HandleSignup)))
	mux.Handle("/api/auth/login", corsMiddleware(http.HandlerFunc(server.HandleLogin)))
	mux.Handle("/api/auth/verify-pin", corsMiddleware(http.HandlerFunc(server.HandleVerifyPIN)))
	mux.Handle("/api/user/profile", corsMiddleware(server.AuthMiddleware(http.HandlerFunc(server.HandleUserProfile))))
	
	// 정적 파일 서빙
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./frontend/js"))))
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./frontend/css"))))
	
	// 루트 핸들러
	fs := http.FileServer(http.Dir("./frontend"))
	mux.Handle("/", fs)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("🌐 HTTP 서버 시작: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
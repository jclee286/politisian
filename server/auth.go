package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  "http://localhost:8080/api/auth/google/callback",
	ClientID:     "", // Set via environment variable
	ClientSecret: "", // Set via environment variable
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	Endpoint:     google.Endpoint,
}

// SessionStore는 세션 토큰과 이메일을 안전하게 매핑하는 서버 측 세션 저장소입니다.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string
}

func (s *SessionStore) Set(token, email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = email
}

func (s *SessionStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	email, exists := s.sessions[token]
	return email, exists
}

var sessionStore = &SessionStore{
	sessions: make(map[string]string),
}

type contextKey string

const userEmailKey contextKey = "email"

// InitOauth 비공개: OAuth 설정을 초기화합니다.
func InitOauth(googleClientID, googleClientSecret string) {
	googleOauthConfig.ClientID = googleClientID
	googleOauthConfig.ClientSecret = googleClientSecret
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// CSRF 공격 방지를 위한 state 생성
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, googleOauthConfig.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != oauthState.Value {
		log.Println("invalid oauth google state")
		http.Error(w, "Invalid oauth state", http.StatusBadRequest)
		return
	}

	data, err := GetUserDataFromGoogle(r.FormValue("code"))
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Error getting user info from Google", http.StatusInternalServerError)
		return
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(data, &userInfo); err != nil {
		http.Error(w, "Failed to unmarshal user info", http.StatusInternalServerError)
		return
	}
	email, ok := userInfo["email"].(string)
	if !ok {
		http.Error(w, "Email not found in user info", http.StatusInternalServerError)
		return
	}

	exists, err := checkUserExists(email)
	if err != nil {
		http.Error(w, fmt.Sprintf("블록체인 사용자 확인 실패: %v", err), http.StatusInternalServerError)
		return
	}

	sessionToken := uuid.New().String()
	// 세션 저장소에 토큰과 이메일을 기록합니다.
	sessionStore.Set(sessionToken, email)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// oauthstate 쿠키는 더 이상 필요 없으므로 삭제
	http.SetCookie(w, &http.Cookie{
		Name:   "oauthstate",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	if exists {
		http.Redirect(w, r, "/index.html", http.StatusSeeOther)
	} else {
		// 새 사용자의 경우, 세션 토큰에 이메일을 잠시 저장해두었다가
		// 프로필 저장 시 사용합니다. 이 부분은 핸들러에서 직접 처리해야 합니다.
		// 이 예제에서는 단순화를 위해 바로 프로필 페이지로 리디렉션합니다.
		http.Redirect(w, r, "/profile.html", http.StatusSeeOther)
	}
}

func GetUserDataFromGoogle(code string) ([]byte, error) {
	token, err := googleOauthConfig.Exchange(context.Background(), code)
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

	return contents, nil
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sessionToken := sessionCookie.Value
		email, exists := sessionStore.Get(sessionToken)
		if !exists {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userEmailKey, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// checkUserExists 함수는 블록체인 쿼리를 수행하여 사용자의 존재 여부를 확인합니다.
func checkUserExists(email string) (bool, error) {
	queryData := []byte(email)
	res, err := blockchainClient.ABCIQuery(context.Background(), "/account/exists", queryData)
	if err != nil {
		return false, fmt.Errorf("ABCIQuery failed: %w", err)
	}

	if res.Response.Code == 0 {
		return true, nil
	}
	
	return false, nil
}

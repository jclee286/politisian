package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	googleOauth2 "google.golang.org/api/oauth2/v2"
)

type contextKey string
const userEmailKey contextKey = "email"

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				http.Error(w, "인증이 필요합니다. 로그인해주세요.", http.StatusUnauthorized)
				return
			}
			http.Error(w, "잘못된 요청입니다.", http.StatusBadRequest)
			return
		}

		email := cookie.Value
		if email == "" {
			http.Error(w, "잘못된 세션입니다.", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userEmailKey, email)
		next.ServeHTTP(w, r.WithContext(ctx))
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
		Path:    "/",
	}
	http.SetCookie(w, &cookie)

	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie("oauthstate")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if r.FormValue("state") != state.Value {
		http.Error(w, "유효하지 않은 oauth state", http.StatusBadRequest)
		return
	}

	data, err := GetUserDataFromGoogle(r.FormValue("code"))
	if err != nil {
		http.Error(w, "Google로부터 사용자 정보 얻기 실패", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    data.Email,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600, 
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:   "oauthstate",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	exists, err := checkUserExists(data.Email)
	if err != nil {
		http.Error(w, "사용자 확인 중 오류가 발생했습니다.", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Redirect(w, r, "/index.html", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/profile.html", http.StatusSeeOther)
	}
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
	if err := json.Unmarshal(contents, &userInfo); err != nil {
		return nil, fmt.Errorf("failed unmarshalling user info: %v", err)
	}

	return &userInfo, nil
}

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

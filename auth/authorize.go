package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"time"
)

func (m *Manager) Authorize(username string) error {
	log := m.logger()
	redirectURI := "http://localhost:8080/callback"

	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=chat:read+chat:edit",
		m.ClientID,
		url.QueryEscape(redirectURI),
	)

	codeCh := make(chan string)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		fmt.Fprintln(w, "Authorization successful. You can close this window.")
		log.Info("oauth callback received", "user", username)
		codeCh <- code
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("oauth callback server stopped", "error", err)
		}
	}()

	// Open browser automatically (works on mac/linux; Windows needs start)
	if err := exec.Command("xdg-open", authURL).Start(); err != nil {
		log.Warn("failed to open browser automatically", "error", err)
	}

	log.Info("complete oauth authorization in browser", "url", authURL, "user", username)

	code := <-codeCh
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Warn("oauth callback server shutdown failed", "error", err)
	}

	return m.exchangeCode(code, redirectURI)
}

func (m *Manager) exchangeCode(code, redirectURI string) error {
	form := url.Values{}
	form.Set("client_id", m.ClientID)
	form.Set("client_secret", m.ClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("auth failed: %d", resp.StatusCode)
	}

	var r struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}

	tok := &Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		ExpiresAt:    time.Now().Unix() + r.ExpiresIn,
	}

	return m.save(tok)
}

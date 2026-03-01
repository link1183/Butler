package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type Manager struct {
	ClientID     string
	ClientSecret string
	File         string
	Logger       *slog.Logger
}

func (m *Manager) GetValidAccessToken() (string, error) {
	tok, err := m.load()
	if err != nil {
		return "", err
	}

	// Refresh 60 seconds before expiry
	if time.Now().Unix() > tok.ExpiresAt-60 {
		m.logger().Info("refreshing access token")
		tok, err := m.refresh(tok.RefreshToken)
		if err != nil {
			return "", err
		}
		if err := m.save(tok); err != nil {
			return "", err
		}
	}

	return tok.AccessToken, nil
}

func (m *Manager) load() (*Token, error) {
	data, err := os.ReadFile(m.File)
	if err != nil {
		return nil, err
	}
	var tok Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func (m *Manager) save(tok *Token) error {
	data, err := json.MarshalIndent(tok, "", "	")
	if err != nil {
		return err
	}
	return os.WriteFile(m.File, data, 0o600)
}

func (m *Manager) refresh(refreshToken string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", m.ClientID)
	form.Set("client_secret", m.ClientSecret)

	resp, err := http.PostForm(
		"https://id.twitch.tv/oauth2/token",
		form,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh failed: %s", body)
	}

	var r struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return &Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		ExpiresAt:    time.Now().Unix() + r.ExpiresIn,
	}, nil
}

func (m *Manager) logger() *slog.Logger {
	if m.Logger != nil {
		return m.Logger
	}

	return slog.Default()
}

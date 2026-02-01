package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type StoredAuth struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Phone        string    `json:"phone"`
	UserID       string    `json:"user_id"`
	FullName     string    `json:"full_name"`
}

func getAuthFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".equishare", "auth.json"), nil
}

func Save(auth *StoredAuth) error {
	path, err := getAuthFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Load() (*StoredAuth, error) {
	path, err := getAuthFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var auth StoredAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

func Clear() error {
	path, err := getAuthFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func IsLoggedIn() bool {
	auth, err := Load()
	if err != nil || auth == nil {
		return false
	}
	return auth.AccessToken != "" && time.Now().Before(auth.ExpiresAt)
}

func GetToken() string {
	auth, err := Load()
	if err != nil || auth == nil {
		return ""
	}
	return auth.AccessToken
}

func GetRefreshToken() string {
	auth, err := Load()
	if err != nil || auth == nil {
		return ""
	}
	return auth.RefreshToken
}

package spotify

import (
	"net/http"

	"github.com/SrVariable/SPL/auth"
)

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

func GetUser(at *auth.AccessToken) (*User, error) {
	var user User
	if err := doJSON(at, http.MethodGet, "https://api.spotify.com/v1/me", &user); err != nil {
		return nil, err
	}

	return &user, nil
}

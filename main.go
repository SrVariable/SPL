package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/SrVariable/SPL/auth"
	"github.com/SrVariable/SPL/config"
)

func generateCodeVerifier(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))

	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func main() {
	env, err := config.NewEnv()
	if err != nil {
		return
	}

	codeVerifier, err := generateCodeVerifier(60)
	if err != nil {
		return
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	userAuthParams := auth.UserAuthParams{
		ClientID: env.ClientId,
		ResponseType: "code",
		RedirectURI: env.RedirectURI,
		State: "helloworld",
		Scope: "user-read-private user-read-email user-read-playback-state playlist-modify-public playlist-modify-private playlist-read-private",
		CodeChallengeMethod: "S256",
		CodeChallenge: codeChallenge,
	}

	accessToken, err := auth.GetAccessToken(codeVerifier, userAuthParams)
	if err != nil {
		return
	}

	fmt.Println(accessToken)
}

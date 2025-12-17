package tools

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func GenerateCode(n int) (string, error ) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func GenerateCodeVerifier() (string, error) {
	return GenerateCode(80)
}

func GenerateCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))

	return base64.RawURLEncoding.EncodeToString(hash[:])
}

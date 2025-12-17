package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	ClientId string
	ClientSecret string
	RedirectURI string
}

func NewEnv() (*Env, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Couldn't load .env file")
	}

	clientId := os.Getenv("CLIENT_ID")
	if clientId == "" {
		return nil, errors.New("CLIENT_ID not set")
	}

	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientSecret == "" {
		return nil, errors.New("CLIENT_SECRET not set")
	}

	redirectURI := os.Getenv("REDIRECT_URI")
	if redirectURI == "" {
		return nil, errors.New("REDIRECT_URI not set")
	}

	return &Env{
		ClientId: clientId,
		ClientSecret: clientSecret,
		RedirectURI: redirectURI,
	}, nil
}

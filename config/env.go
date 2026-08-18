package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// DSN builds the connection string expected by the go-sql-driver/mysql driver.
// utf8mb4 is required to store track and artist names containing emoji or
// non-latin characters, and parseTime lets DATETIME columns decode into time.Time.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&loc=Local",
		url.QueryEscape(c.User),
		url.QueryEscape(c.Password),
		c.Host,
		c.Port,
		c.Name,
	)
}

type Env struct {
	ClientId    string
	RedirectURI string
	DB          DBConfig
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func NewEnv() (*Env, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Couldn't load .env file")
	}

	clientId := os.Getenv("CLIENT_ID")
	if clientId == "" {
		return nil, errors.New("CLIENT_ID not set")
	}

	redirectURI := os.Getenv("REDIRECT_URI")
	if redirectURI == "" {
		return nil, errors.New("REDIRECT_URI not set")
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return nil, errors.New("DB_USER not set")
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return nil, errors.New("DB_PASSWORD not set")
	}

	return &Env{
		ClientId:    clientId,
		RedirectURI: redirectURI,
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     dbUser,
			Password: dbPassword,
			Name:     getEnv("DB_NAME", "spl"),
		},
	}, nil
}

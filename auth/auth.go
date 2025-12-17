package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/browser"
)

type UserAuth struct {
	Code string `json:"code,omitempty"`
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}
type UserAuthParams struct {
	ClientID string
	ResponseType string
	RedirectURI string
	State string
	Scope string
	CodeChallengeMethod string
	CodeChallenge string
}

func buildUserAuthURL(userAuthParams UserAuthParams) (*url.URL, error) {
	u, err := url.Parse("https://accounts.spotify.com/authorize")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("client_id", userAuthParams.ClientID)
	q.Set("response_type", userAuthParams.ResponseType)
	q.Set("redirect_uri", userAuthParams.RedirectURI)
	q.Set("state", userAuthParams.State)
	q.Set("scope", userAuthParams.Scope)
	q.Set("code_challenge_method", userAuthParams.CodeChallengeMethod)
	q.Set("code_challenge", userAuthParams.CodeChallenge)

	u.RawQuery = q.Encode()

	return u, nil
}

func getUserAuth(userAuthParams UserAuthParams) (*UserAuth, error) {
	userAuthCh := make(chan *UserAuth)
	go func() {
		userAuth, _ := WaitForAuthCode(userAuthParams.State)
		userAuthCh <- userAuth
	}()

	u, err := buildUserAuthURL(userAuthParams)
	if err != nil {
		return nil, err
	}
	browser.OpenURL(u.String())

	userAuth := <-userAuthCh
	if userAuth.Error != "" {
		return nil, fmt.Errorf(userAuth.Error)
	}

	return userAuth, nil
}

type AccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType string `json:"token_type"`
	Scope string `json:"scope"`
	ExpiresIn int `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}
type AccessTokenParams struct {
	GrantType string
	Code string
	RedirectURI string
	ClientID string
	CodeVerifier string
}

func GetAccessToken(codeVerifier string, userAuthParams UserAuthParams) (*AccessToken, error) {
	userAuth, err := getUserAuth(userAuthParams)
	if err != nil {
		return nil, err
	}

	accessTokenParams := AccessTokenParams{
		GrantType: "authorization_code",
		Code: userAuth.Code,
		RedirectURI: userAuthParams.RedirectURI,
		ClientID: userAuthParams.ClientID,
		CodeVerifier: codeVerifier,
	}

	endpoint := "https://accounts.spotify.com/api/token"

	data := url.Values{}
	data.Set("grant_type", accessTokenParams.GrantType)
	data.Set("code", accessTokenParams.Code)
	data.Set("redirect_uri", accessTokenParams.RedirectURI)
	data.Set("client_id", accessTokenParams.ClientID)
	data.Set("code_verifier", accessTokenParams.CodeVerifier)

	req, err := http.NewRequest(
		http.MethodPost,
		endpoint,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var accessToken AccessToken
	if err := json.NewDecoder(resp.Body).Decode(&accessToken); err != nil {
		return nil, err
	}

	return &accessToken, nil
}

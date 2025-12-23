package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
)

/* Reference: https://developer.spotify.com/documentation/web-api/tutorials/code-pkce-flow#response */
type UserAuth struct {
	Code  string `json:"code,omitempty"`
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

/* Reference: https://developer.spotify.com/documentation/web-api/tutorials/code-pkce-flow#request-user-authorization */
type UserAuthParams struct {
	ClientID            string
	ResponseType        string
	RedirectURI         string
	State               string
	Scope               string
	CodeChallengeMethod string
	CodeChallenge       string
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

/* Reference: https://developer.spotify.com/documentation/web-api/tutorials/code-pkce-flow#response-1 */
type AccessToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`

	// To calculate when the token expires
	CreatedAt time.Time `json:"created_at"`
	ClientID  string    `json:"client_id"`
}

/* Reference: https://developer.spotify.com/documentation/web-api/tutorials/code-pkce-flow#request-an-access-token */
type AccessTokenParams struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	CodeVerifier string
}

const filename = "token.json"

func (at *AccessToken) Save() error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	file.Chmod(0600)
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(at); err != nil {
		return err
	}

	return nil
}

func (at *AccessToken) isExpired() bool {
	expiration := at.CreatedAt.Add(time.Duration(at.ExpiresIn) * time.Second)
	return time.Now().After(expiration)
}

type RefreshTokenParams struct {
	GrantType    string
	RefreshToken string
	ClientID     string
}

func refreshToken(at AccessToken) (*AccessToken, error) {
	refreshTokenParams := RefreshTokenParams{
		GrantType:    "refresh_token",
		RefreshToken: at.RefreshToken,
		ClientID:     at.ClientID,
	}

	data := url.Values{}
	data.Set("grant_type", refreshTokenParams.GrantType)
	data.Set("refresh_token", refreshTokenParams.RefreshToken)
	data.Set("client_id", refreshTokenParams.ClientID)

	endpoint := "https://accounts.spotify.com/api/token"
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

	accessToken.CreatedAt = time.Now()
	accessToken.ClientID = at.ClientID

	return &accessToken, nil
}

func loadAccessToken() (*AccessToken, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var accessToken AccessToken
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&accessToken); err != nil {
		return nil, err
	}

	if accessToken.isExpired() {
		fmt.Println("Token expired, refreshing token...")
		return refreshToken(accessToken)
	}

	return &accessToken, nil
}

func GetAccessToken(codeVerifier string, userAuthParams UserAuthParams) (*AccessToken, error) {
	at, err := loadAccessToken()
	if at != nil {
		return at, nil
	}
	fmt.Println(err)

	fmt.Println("Generating new access token...")
	userAuth, err := getUserAuth(userAuthParams)
	if err != nil {
		return nil, err
	}

	accessTokenParams := AccessTokenParams{
		GrantType:    "authorization_code",
		Code:         userAuth.Code,
		RedirectURI:  userAuthParams.RedirectURI,
		ClientID:     userAuthParams.ClientID,
		CodeVerifier: codeVerifier,
	}

	data := url.Values{}
	data.Set("grant_type", accessTokenParams.GrantType)
	data.Set("code", accessTokenParams.Code)
	data.Set("redirect_uri", accessTokenParams.RedirectURI)
	data.Set("client_id", accessTokenParams.ClientID)
	data.Set("code_verifier", accessTokenParams.CodeVerifier)

	endpoint := "https://accounts.spotify.com/api/token"
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
	accessToken.CreatedAt = time.Now()
	accessToken.ClientID = userAuthParams.ClientID

	return &accessToken, nil
}

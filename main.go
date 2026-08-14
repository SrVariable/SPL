package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SrVariable/SPL/auth"
	"github.com/SrVariable/SPL/config"
	"github.com/SrVariable/SPL/tools"
)

type User struct {
	ID string `json:"id"`
	DisplayName string `json:"display_name"`
}

func GetUser(at *auth.AccessToken) (*User, error) {
	endpoint := "https://api.spotify.com/v1/me"
	req, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

type PlaylistTracks struct {
	Total int `json:"total"`
}
type Playlist struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Tracks PlaylistTracks `json:"tracks"`
}

type PlaylistResponse struct {
	Items []Playlist `json:"items"`
}

func (u *User) GetPlaylists(at *auth.AccessToken) ([]Playlist, error) {
	endpoint := "https://api.spotify.com/v1/me/playlists"
	req, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil,
	)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result PlaylistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (u *User) SelectPlaylist(at *auth.AccessToken) {
	playlists, err := u.GetPlaylists(at)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Your playlists")
	for i, playlist := range playlists {
		fmt.Println(i + 1, playlist.Name, playlist.Tracks.Total)
	}
	fmt.Print("Select the playlist: ")
}

func main() {
	env, err := config.NewEnv()
	if err != nil {
		fmt.Println(err)
		return
	}

	state, err := tools.GenerateCode(60)
	if err != nil {
		fmt.Println(err)
		return
	}

	codeVerifier, err := tools.GenerateCodeVerifier()
	if err != nil {
		fmt.Println(err)
		return
	}

	codeChallenge := tools.GenerateCodeChallenge(codeVerifier)

	userAuthParams := auth.UserAuthParams{
		ClientID:            env.ClientId,
		ResponseType:        "code",
		RedirectURI:         env.RedirectURI,
		State:               state,
		Scope:               "user-read-private user-read-email user-read-playback-state playlist-modify-public playlist-modify-private playlist-read-private",
		CodeChallengeMethod: "S256",
		CodeChallenge:       codeChallenge,
	}

	accessToken, err := auth.GetAccessToken(codeVerifier, userAuthParams)
	if err != nil {
		fmt.Println(err)
		return
	}

	accessToken.Save()

	user, err := GetUser(accessToken)
	if err != nil {
		fmt.Println(err)
		return
	}

	user.SelectPlaylist(accessToken)
}

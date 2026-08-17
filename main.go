package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/SrVariable/SPL/auth"
	"github.com/SrVariable/SPL/config"
	"github.com/SrVariable/SPL/spotify"
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

// splitLinks splits a line of input into individual links, allowing the
// user to separate multiple links with spaces, commas, tabs or newlines.
func splitLinks(line string) []string {
	return strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
}

func addSongsToQueue(scanner *bufio.Scanner, at *auth.AccessToken) {
	fmt.Println("Paste one or more Spotify track links (separated by spaces, commas or new lines).")
	fmt.Println("Enter an empty line when you're done.")

	var links []string
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		links = append(links, splitLinks(line)...)
	}

	if len(links) == 0 {
		fmt.Println("No links provided")
		return
	}

	for _, link := range links {
		trackID, err := spotify.ParseTrackID(link)
		if err != nil {
			fmt.Printf("Skipping %q: %v\n", link, err)
			continue
		}

		if err := spotify.AddToQueue(at, trackID); err != nil {
			fmt.Printf("Failed to queue %s: %v\n", trackID, err)
			continue
		}

		fmt.Printf("Queued track %s\n", trackID)
	}
}

func showMenu() {
	fmt.Println("\nWhat would you like to do?")
	fmt.Println("1. Add song(s) to queue")
	fmt.Println("2. Select playlist")
	fmt.Println("3. Exit")
	fmt.Print("> ")
}

func run(user *User, accessToken *auth.AccessToken) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		showMenu()
		if !scanner.Scan() {
			return
		}

		switch strings.TrimSpace(scanner.Text()) {
		case "1":
			addSongsToQueue(scanner, accessToken)
		case "2":
			user.SelectPlaylist(accessToken)
		case "3":
			fmt.Println("Bye!")
			return
		default:
			fmt.Println("Invalid option")
		}
	}
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
		Scope:               "user-read-private user-read-email user-read-playback-state user-modify-playback-state playlist-modify-public playlist-modify-private playlist-read-private",
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

	run(user, accessToken)
}

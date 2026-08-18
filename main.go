package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/SrVariable/SPL/auth"
	"github.com/SrVariable/SPL/config"
	"github.com/SrVariable/SPL/db"
	"github.com/SrVariable/SPL/spotify"
	"github.com/SrVariable/SPL/tools"
)

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

// selectPlaylist lists the user's playlists and lets them answer with either a
// number from the list or a pasted playlist link.
func selectPlaylist(scanner *bufio.Scanner, at *auth.AccessToken) (*spotify.Playlist, error) {
	playlists, err := spotify.GetPlaylists(at)
	if err != nil {
		return nil, err
	}

	fmt.Println("\nYour playlists:")
	for i, playlist := range playlists {
		fmt.Printf("%3d. %s (%d tracks)\n", i+1, playlist.Name, playlist.Tracks.Total)
	}

	fmt.Print("Select a playlist by number, or paste its link: ")
	if !scanner.Scan() {
		return nil, fmt.Errorf("no playlist selected")
	}
	answer := strings.TrimSpace(scanner.Text())

	if choice, err := strconv.Atoi(answer); err == nil {
		if choice < 1 || choice > len(playlists) {
			return nil, fmt.Errorf("%d is not in the list", choice)
		}

		return &playlists[choice-1], nil
	}

	playlistID, err := spotify.ParsePlaylistID(answer)
	if err != nil {
		return nil, err
	}

	return spotify.GetPlaylist(at, playlistID)
}

func savePlaylist(scanner *bufio.Scanner, at *auth.AccessToken, database *sql.DB) {
	playlist, err := selectPlaylist(scanner, at)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Fetching %q...\n", playlist.Name)
	tracks, skipped, err := spotify.GetPlaylistTracks(at, playlist.ID, func(fetched, total int) {
		fmt.Printf("\r  %d/%d", fetched, total)
	})
	fmt.Println()
	if err != nil {
		fmt.Println(err)
		return
	}

	stats, err := db.SyncPlaylist(context.Background(), database, *playlist, tracks, skipped)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Saved %d tracks by %d artists", stats.Tracks, stats.Artists)
	if stats.Skipped > 0 {
		fmt.Printf(" (%d entries skipped: local files, unavailable tracks or episodes)", stats.Skipped)
	}
	fmt.Println()
}

func showMenu() {
	fmt.Println("\nWhat would you like to do?")
	fmt.Println("1. Add song(s) to queue")
	fmt.Println("2. Save a playlist to the database")
	fmt.Println("3. Exit")
	fmt.Print("> ")
}

func run(accessToken *auth.AccessToken, database *sql.DB) {
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
			savePlaylist(scanner, accessToken, database)
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

	database, err := db.Connect(env.DB)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer database.Close()

	if err := db.Migrate(context.Background(), database); err != nil {
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

	user, err := spotify.GetUser(accessToken)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Logged in as %s\n", user.DisplayName)

	run(accessToken, database)
}

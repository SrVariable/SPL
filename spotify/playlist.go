package spotify

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/SrVariable/SPL/auth"
)

type ExternalURLs struct {
	Spotify string `json:"spotify"`
}

type Album struct {
	Name string `json:"name"`
}

type Owner struct {
	DisplayName string `json:"display_name"`
}

type TrackCount struct {
	Total int `json:"total"`
}

type Artist struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

func (a Artist) URL() string {
	if a.ExternalURLs.Spotify != "" {
		return a.ExternalURLs.Spotify
	}

	return fmt.Sprintf("https://open.spotify.com/artist/%s", a.ID)
}

type Track struct {
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	Name         string       `json:"name"`
	DurationMs   int          `json:"duration_ms"`
	Album        Album        `json:"album"`
	Artists      []Artist     `json:"artists"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

func (t Track) URL() string {
	if t.ExternalURLs.Spotify != "" {
		return t.ExternalURLs.Spotify
	}

	return fmt.Sprintf("https://open.spotify.com/track/%s", t.ID)
}

type Playlist struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	SnapshotID   string       `json:"snapshot_id"`
	Owner        Owner        `json:"owner"`
	ExternalURLs ExternalURLs `json:"external_urls"`
	Tracks       TrackCount   `json:"tracks"`
}

// PlaylistTrack is one entry of a playlist. Track is a pointer because Spotify
// returns null for tracks that are no longer available in the catalogue.
type PlaylistTrack struct {
	AddedAt time.Time `json:"added_at"`
	IsLocal bool      `json:"is_local"`
	Track   *Track    `json:"track"`

	// Position is the index inside the playlist, counting the entries we skip.
	Position int `json:"-"`
}

// page is the paging object every list endpoint wraps its results in. Next is
// the ready-to-use URL of the following page, empty once there are no more.
type page[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next"`
	Total int    `json:"total"`
}

// GetPlaylists returns every playlist of the current user, following the paging
// links instead of computing offsets by hand.
func GetPlaylists(at *auth.AccessToken) ([]Playlist, error) {
	endpoint := "https://api.spotify.com/v1/me/playlists?limit=50"

	var playlists []Playlist
	for endpoint != "" {
		var result page[Playlist]
		if err := doJSON(at, http.MethodGet, endpoint, &result); err != nil {
			return nil, err
		}

		// Spotify occasionally returns null entries in this list.
		for _, playlist := range result.Items {
			if playlist.ID != "" {
				playlists = append(playlists, playlist)
			}
		}

		endpoint = result.Next
	}

	return playlists, nil
}

const playlistFields = "id,name,snapshot_id,external_urls(spotify),owner(display_name),tracks(total)"

func GetPlaylist(at *auth.AccessToken, playlistID string) (*Playlist, error) {
	endpoint, err := url.Parse(fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", url.PathEscape(playlistID)))
	if err != nil {
		return nil, err
	}

	q := endpoint.Query()
	q.Set("fields", playlistFields)
	endpoint.RawQuery = q.Encode()

	var playlist Playlist
	if err := doJSON(at, http.MethodGet, endpoint.String(), &playlist); err != nil {
		return nil, err
	}

	return &playlist, nil
}

// trackFields trims the response down to what we store. Playlist items are by
// far the heaviest objects in the API, and a big playlist means many pages.
const trackFields = "next,total,items(added_at,is_local," +
	"track(id,type,name,duration_ms,external_urls(spotify),album(name)," +
	"artists(id,name,external_urls(spotify))))"

// keep reports whether a playlist entry is something we can store: local files
// have no id, unavailable tracks come back as null, and podcast episodes have
// no artists.
func (pt PlaylistTrack) keep() bool {
	return pt.Track != nil && !pt.IsLocal && pt.Track.ID != "" && pt.Track.Type != "episode"
}

// GetPlaylistTracks walks every page of a playlist and returns the entries worth
// storing, along with how many were skipped. progress may be nil.
func GetPlaylistTracks(at *auth.AccessToken, playlistID string, progress func(fetched, total int)) ([]PlaylistTrack, int, error) {
	endpoint, err := url.Parse(fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", url.PathEscape(playlistID)))
	if err != nil {
		return nil, 0, err
	}

	q := endpoint.Query()
	q.Set("fields", trackFields)
	q.Set("limit", "50")
	endpoint.RawQuery = q.Encode()

	var (
		tracks  []PlaylistTrack
		next    = endpoint.String()
		skipped int
		seen    int
	)
	for next != "" {
		var result page[PlaylistTrack]
		if err := doJSON(at, http.MethodGet, next, &result); err != nil {
			return nil, 0, err
		}

		for _, item := range result.Items {
			item.Position = seen
			seen++

			if !item.keep() {
				skipped++
				continue
			}

			tracks = append(tracks, item)
		}

		if progress != nil {
			progress(seen, result.Total)
		}

		next = result.Next
	}

	return tracks, skipped, nil
}

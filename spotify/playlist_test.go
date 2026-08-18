package spotify

import (
	"encoding/json"
	"testing"
)

func TestParsePlaylistID(t *testing.T) {
	cases := map[string]string{
		"https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M":              "37i9dQZF1DXcBWIGoYBM5M",
		"https://open.spotify.com/intl-es/playlist/37i9dQZF1DXcBWIGoYBM5M":      "37i9dQZF1DXcBWIGoYBM5M",
		"https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M?si=abc123def": "37i9dQZF1DXcBWIGoYBM5M",
	}

	for link, want := range cases {
		got, err := ParsePlaylistID(link)
		if err != nil {
			t.Fatalf("ParsePlaylistID(%q) returned error: %v", link, err)
		}
		if got != want {
			t.Errorf("ParsePlaylistID(%q) = %q, want %q", link, got, want)
		}
	}

	// A track link must not be accepted as a playlist link.
	if _, err := ParsePlaylistID("https://open.spotify.com/track/6yRN1GztxFYi1Dk1Pv0qSQ"); err == nil {
		t.Error("expected an error for a track link, got nil")
	}
}

// A real playlist page mixes regular tracks with entries we can't store: local
// files, tracks pulled from the catalogue (null) and podcast episodes.
const playlistPage = `{
  "total": 4,
  "next": null,
  "items": [
    {
      "added_at": "2023-04-01T10:00:00Z",
      "is_local": false,
      "track": {
        "id": "6yRN1GztxFYi1Dk1Pv0qSQ",
        "type": "track",
        "name": "Song",
        "duration_ms": 210000,
        "external_urls": { "spotify": "https://open.spotify.com/track/6yRN1GztxFYi1Dk1Pv0qSQ" },
        "album": { "name": "Album" },
        "artists": [
          { "id": "1", "name": "Main", "external_urls": { "spotify": "https://open.spotify.com/artist/1" } },
          { "id": "2", "name": "Feat", "external_urls": { "spotify": "https://open.spotify.com/artist/2" } }
        ]
      }
    },
    { "added_at": "2023-04-02T10:00:00Z", "is_local": false, "track": null },
    {
      "added_at": "2023-04-03T10:00:00Z",
      "is_local": true,
      "track": { "id": null, "type": "track", "name": "Local file", "artists": [] }
    },
    {
      "added_at": "2023-04-04T10:00:00Z",
      "is_local": false,
      "track": { "id": "abc", "type": "episode", "name": "Episode", "artists": [] }
    }
  ]
}`

func TestPlaylistPageKeepsOnlyStorableTracks(t *testing.T) {
	var page page[PlaylistTrack]
	if err := json.Unmarshal([]byte(playlistPage), &page); err != nil {
		t.Fatalf("failed to decode the playlist page: %v", err)
	}

	if page.Next != "" {
		t.Errorf("Next = %q, want empty for the last page", page.Next)
	}

	var kept []PlaylistTrack
	for _, item := range page.Items {
		if item.keep() {
			kept = append(kept, item)
		}
	}

	if len(kept) != 1 {
		t.Fatalf("kept %d entries, want 1", len(kept))
	}

	track := kept[0].Track
	if track.Name != "Song" {
		t.Errorf("Name = %q, want %q", track.Name, "Song")
	}
	if len(track.Artists) != 2 {
		t.Errorf("got %d artists, want 2", len(track.Artists))
	}
	if kept[0].AddedAt.IsZero() {
		t.Error("AddedAt wasn't parsed")
	}
}

func TestTrackURLFallsBackToTheID(t *testing.T) {
	track := Track{ID: "6yRN1GztxFYi1Dk1Pv0qSQ"}
	want := "https://open.spotify.com/track/6yRN1GztxFYi1Dk1Pv0qSQ"
	if got := track.URL(); got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}

	artist := Artist{ID: "1"}
	if got := artist.URL(); got != "https://open.spotify.com/artist/1" {
		t.Errorf("URL() = %q, want %q", got, "https://open.spotify.com/artist/1")
	}
}

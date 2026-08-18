package spotify

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/SrVariable/SPL/auth"
)

// urlPatterns holds one compiled pattern per kind of Spotify link, e.g.
// https://open.spotify.com/intl-es/track/<id>?si=<token>
var urlPatterns = map[string]*regexp.Regexp{
	"track":    urlPattern("track"),
	"playlist": urlPattern("playlist"),
}

func urlPattern(kind string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`open\.spotify\.com/(?:intl-\w+/)?%s/([a-zA-Z0-9]+)`, kind))
}

func parseID(kind, link string) (string, error) {
	matches := urlPatterns[kind].FindStringSubmatch(link)
	if len(matches) < 2 {
		return "", fmt.Errorf("couldn't find a %s id in %q", kind, link)
	}

	return matches[1], nil
}

func ParseTrackID(link string) (string, error) {
	return parseID("track", link)
}

func ParsePlaylistID(link string) (string, error) {
	return parseID("playlist", link)
}

func AddToQueue(at *auth.AccessToken, trackID string) error {
	endpoint, err := url.Parse("https://api.spotify.com/v1/me/player/queue")
	if err != nil {
		return err
	}

	q := endpoint.Query()
	q.Set("uri", fmt.Sprintf("spotify:track:%s", trackID))
	endpoint.RawQuery = q.Encode()

	resp, err := do(at, http.MethodPost, endpoint.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// It should be http.StatusNoContent according to https://developer.spotify.com/documentation/web-api/reference/add-to-queue,
	// but the return value was http.StatusOK
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add track %s to queue: %s", trackID, resp.Status)
	}

	return nil
}

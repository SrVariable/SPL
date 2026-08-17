package spotify

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/SrVariable/SPL/auth"
)

var trackURLPattern = regexp.MustCompile(`open\.spotify\.com/(?:intl-\w+/)?track/([a-zA-Z0-9]+)`)

func ParseTrackID(link string) (string, error) {
	matches := trackURLPattern.FindStringSubmatch(link)
	if len(matches) < 2 {
		return "", fmt.Errorf("couldn't find a track id in %q", link)
	}

	return matches[1], nil
}

func AddToQueue(at *auth.AccessToken, trackID string) error {
	endpoint, err := url.Parse("https://api.spotify.com/v1/me/player/queue")
	if err != nil {
		return err
	}

	q := endpoint.Query()
	q.Set("uri", fmt.Sprintf("spotify:track:%s", trackID))
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at.AccessToken))

	resp, err := http.DefaultClient.Do(req)
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

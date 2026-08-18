package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/SrVariable/SPL/auth"
)

// maxRetries bounds how many times a request is replayed after a 429. Syncing a
// big playlist means dozens of calls in a row, which is exactly when Spotify
// starts rate limiting.
const maxRetries = 3

// retryAfter reads the Retry-After header, falling back to a short pause when
// the header is missing or unparsable.
func retryAfter(resp *http.Response) time.Duration {
	seconds, err := strconv.Atoi(resp.Header.Get("Retry-After"))
	if err != nil || seconds <= 0 {
		return time.Second
	}

	return time.Duration(seconds) * time.Second
}

// do performs an authenticated request against the Web API, retrying on 429.
// The caller owns the response body and must close it.
func do(at *auth.AccessToken, method, endpoint string) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(method, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at.AccessToken))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			wait := retryAfter(resp)
			resp.Body.Close()
			fmt.Printf("Rate limited by Spotify, retrying in %s...\n", wait)
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, fmt.Errorf("the access token was rejected, delete token.json and authorize again")
		}

		return resp, nil
	}
}

// doJSON performs an authenticated request and decodes a successful response
// body into out.
func doJSON(at *auth.AccessToken, method, endpoint string, out any) error {
	resp, err := do(at, method, endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s", method, endpoint, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

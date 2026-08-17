package spotify

import "testing"

func TestParseTrackID(t *testing.T) {
	cases := map[string]string{
		"https://open.spotify.com/intl-es/track/6yRN1GztxFYi1Dk1Pv0qSQ?si=41c9e0d5e06640f5": "6yRN1GztxFYi1Dk1Pv0qSQ",
		"https://open.spotify.com/track/6yRN1GztxFYi1Dk1Pv0qSQ":                              "6yRN1GztxFYi1Dk1Pv0qSQ",
		"https://open.spotify.com/track/6yRN1GztxFYi1Dk1Pv0qSQ?si=abc":                        "6yRN1GztxFYi1Dk1Pv0qSQ",
	}

	for link, want := range cases {
		got, err := ParseTrackID(link)
		if err != nil {
			t.Fatalf("ParseTrackID(%q) returned error: %v", link, err)
		}
		if got != want {
			t.Errorf("ParseTrackID(%q) = %q, want %q", link, got, want)
		}
	}

	if _, err := ParseTrackID("not a spotify link"); err == nil {
		t.Error("expected an error for an invalid link, got nil")
	}
}

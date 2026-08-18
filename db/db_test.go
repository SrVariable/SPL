package db

import (
	"strings"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	statements := splitStatements(schema)

	if len(statements) == 0 {
		t.Fatal("the embedded schema produced no statements")
	}

	for _, statement := range statements {
		if strings.HasPrefix(statement, "--") {
			t.Errorf("comment leaked into a statement: %q", firstLine(statement))
		}
		if strings.Contains(statement, ";") {
			t.Errorf("statement wasn't split properly: %q", firstLine(statement))
		}
	}

	// Every table the sync writes to has to be created by the schema.
	for _, table := range []string{"playlists", "artists", "tracks", "track_artists", "playlist_tracks", "genres", "artist_genres", "track_genres"} {
		want := "CREATE TABLE IF NOT EXISTS " + table + " ("
		if !strings.Contains(schema, want) {
			t.Errorf("the schema doesn't create the %q table", table)
		}
	}
}

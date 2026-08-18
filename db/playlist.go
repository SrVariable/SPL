package db

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/SrVariable/SPL/spotify"
)

// batchSize keeps each multi-row INSERT well below the server's limit on
// placeholders while still cutting the number of round trips.
const batchSize = 500

type SyncStats struct {
	// Tracks is how many distinct tracks ended up linked to the playlist.
	Tracks int
	// Artists is how many distinct artists those tracks belong to.
	Artists int
	// Skipped is how many playlist entries couldn't be stored (local files,
	// unavailable tracks, podcast episodes).
	Skipped int
}

// upsert runs a multi-row INSERT ... ON DUPLICATE KEY UPDATE in chunks, so a
// playlist with thousands of tracks doesn't build one enormous statement.
func upsert(ctx context.Context, tx *sql.Tx, table string, columns []string, onDuplicate string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}

	row := "(" + strings.Join(slices.Repeat([]string{"?"}, len(columns)), ", ") + ")"

	for chunk := range slices.Chunk(rows, batchSize) {
		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*len(columns))
		for _, values := range chunk {
			placeholders = append(placeholders, row)
			args = append(args, values...)
		}

		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s ON DUPLICATE KEY UPDATE %s",
			table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
			onDuplicate,
		)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to upsert into %s: %w", table, err)
		}
	}

	return nil
}

// nullTime maps a missing added_at to SQL NULL; a zero time.Time is not a valid
// DATETIME under strict mode.
func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}

	return t
}

// SyncPlaylist writes a whole playlist in a single transaction. The tracks are
// expected to be already fetched, so no HTTP call happens while the transaction
// is open.
func SyncPlaylist(ctx context.Context, database *sql.DB, playlist spotify.Playlist, items []spotify.PlaylistTrack, skipped int) (SyncStats, error) {
	stats := SyncStats{Skipped: skipped}

	// A playlist may list the same track more than once; collapse those so the
	// statements below don't fight over the same primary key.
	trackRows := map[string][]any{}
	artistRows := map[string][]any{}
	var trackArtistRows, playlistTrackRows [][]any

	for _, item := range items {
		track := item.Track

		if _, ok := trackRows[track.ID]; ok {
			continue
		}
		trackRows[track.ID] = []any{track.ID, track.Name, track.URL(), track.Album.Name, track.DurationMs}
		playlistTrackRows = append(playlistTrackRows, []any{playlist.ID, track.ID, item.Position, nullTime(item.AddedAt)})

		for position, artist := range track.Artists {
			if artist.ID == "" {
				continue
			}
			artistRows[artist.ID] = []any{artist.ID, artist.Name, artist.URL()}
			trackArtistRows = append(trackArtistRows, []any{track.ID, artist.ID, position})
		}
	}

	stats.Tracks = len(trackRows)
	stats.Artists = len(artistRows)

	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return stats, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO playlists (id, name, owner_name, snapshot_id, spotify_url, total_tracks, synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, NOW())
		 ON DUPLICATE KEY UPDATE
		     name = VALUES(name),
		     owner_name = VALUES(owner_name),
		     snapshot_id = VALUES(snapshot_id),
		     spotify_url = VALUES(spotify_url),
		     total_tracks = VALUES(total_tracks),
		     synced_at = VALUES(synced_at)`,
		playlist.ID, playlist.Name, playlist.Owner.DisplayName, playlist.SnapshotID,
		playlist.ExternalURLs.Spotify, playlist.Tracks.Total,
	)
	if err != nil {
		return stats, fmt.Errorf("failed to save playlist %s: %w", playlist.ID, err)
	}

	// Tracks removed from the playlist upstream have to disappear from the dump.
	// The rows in tracks/artists stay: they aren't owned by this playlist.
	if _, err := tx.ExecContext(ctx, "DELETE FROM playlist_tracks WHERE playlist_id = ?", playlist.ID); err != nil {
		return stats, fmt.Errorf("failed to clear playlist %s: %w", playlist.ID, err)
	}

	err = upsert(ctx, tx, "artists",
		[]string{"id", "name", "spotify_url"},
		"name = VALUES(name), spotify_url = VALUES(spotify_url)",
		slices.Collect(maps.Values(artistRows)),
	)
	if err != nil {
		return stats, err
	}

	err = upsert(ctx, tx, "tracks",
		[]string{"id", "name", "spotify_url", "album", "duration_ms"},
		"name = VALUES(name), spotify_url = VALUES(spotify_url), album = VALUES(album), duration_ms = VALUES(duration_ms)",
		slices.Collect(maps.Values(trackRows)),
	)
	if err != nil {
		return stats, err
	}

	err = upsert(ctx, tx, "track_artists",
		[]string{"track_id", "artist_id", "position"},
		"position = VALUES(position)",
		trackArtistRows,
	)
	if err != nil {
		return stats, err
	}

	err = upsert(ctx, tx, "playlist_tracks",
		[]string{"playlist_id", "track_id", "position", "added_at"},
		"position = VALUES(position), added_at = VALUES(added_at)",
		playlistTrackRows,
	)
	if err != nil {
		return stats, err
	}

	if err := tx.Commit(); err != nil {
		return stats, err
	}

	return stats, nil
}

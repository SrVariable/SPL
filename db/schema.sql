-- Spotify base62 ids are 22 characters long; VARCHAR(32) leaves some room.
-- Every table is utf8mb4 so track and artist names keep their accents and emoji.

CREATE TABLE IF NOT EXISTS playlists (
	id           VARCHAR(32)  NOT NULL,
	name         VARCHAR(255) NOT NULL,
	owner_name   VARCHAR(255)     NULL,
	snapshot_id  VARCHAR(255)     NULL,
	spotify_url  VARCHAR(255)     NULL,
	total_tracks INT          NOT NULL DEFAULT 0,
	synced_at    DATETIME         NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS artists (
	id          VARCHAR(32)  NOT NULL,
	name        VARCHAR(255) NOT NULL,
	spotify_url VARCHAR(255)     NULL,
	-- NULL means the genres for this artist have never been resolved.
	genres_synced_at DATETIME    NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tracks (
	id          VARCHAR(32)  NOT NULL,
	name        VARCHAR(255) NOT NULL,
	spotify_url VARCHAR(255) NOT NULL,
	album       VARCHAR(255)     NULL,
	duration_ms INT              NULL,
	created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- position 0 is the main artist, the rest are featurings in the order Spotify returns them.
CREATE TABLE IF NOT EXISTS track_artists (
	track_id  VARCHAR(32)      NOT NULL,
	artist_id VARCHAR(32)      NOT NULL,
	position  TINYINT UNSIGNED NOT NULL DEFAULT 0,
	PRIMARY KEY (track_id, artist_id),
	KEY idx_track_artists_artist (artist_id),
	CONSTRAINT fk_track_artists_track  FOREIGN KEY (track_id)  REFERENCES tracks(id)  ON DELETE CASCADE,
	CONSTRAINT fk_track_artists_artist FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- A playlist may contain the same track more than once; the primary key collapses
-- those into a single row, which is what we want when classifying by style.
CREATE TABLE IF NOT EXISTS playlist_tracks (
	playlist_id VARCHAR(32) NOT NULL,
	track_id    VARCHAR(32) NOT NULL,
	position    INT         NOT NULL,
	added_at    DATETIME        NULL,
	PRIMARY KEY (playlist_id, track_id),
	KEY idx_playlist_tracks_track (track_id),
	CONSTRAINT fk_playlist_tracks_playlist FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
	CONSTRAINT fk_playlist_tracks_track    FOREIGN KEY (track_id)    REFERENCES tracks(id)    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Spotify only exposes genres per artist, never per track, and for many artists the
-- list comes back empty. These three tables stay empty for now and are the place to
-- fill genres in later, either from /v1/artists or by hand.
CREATE TABLE IF NOT EXISTS genres (
	id   INT UNSIGNED NOT NULL AUTO_INCREMENT,
	name VARCHAR(120) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_genres_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS artist_genres (
	artist_id VARCHAR(32)  NOT NULL,
	genre_id  INT UNSIGNED NOT NULL,
	PRIMARY KEY (artist_id, genre_id),
	KEY idx_artist_genres_genre (genre_id),
	CONSTRAINT fk_artist_genres_artist FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE,
	CONSTRAINT fk_artist_genres_genre  FOREIGN KEY (genre_id)  REFERENCES genres(id)  ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS track_genres (
	track_id VARCHAR(32)  NOT NULL,
	genre_id INT UNSIGNED NOT NULL,
	source   ENUM('artist', 'manual') NOT NULL DEFAULT 'manual',
	PRIMARY KEY (track_id, genre_id),
	KEY idx_track_genres_genre (genre_id),
	CONSTRAINT fk_track_genres_track FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE,
	CONSTRAINT fk_track_genres_genre FOREIGN KEY (genre_id) REFERENCES genres(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Flattened view for browsing from any SQL client.
CREATE OR REPLACE VIEW v_playlist_tracks AS
SELECT
	p.id   AS playlist_id,
	p.name AS playlist,
	pt.position,
	t.id   AS track_id,
	t.name AS track,
	(
		SELECT GROUP_CONCAT(a.name ORDER BY ta.position SEPARATOR ', ')
		FROM track_artists ta
		JOIN artists a ON a.id = ta.artist_id
		WHERE ta.track_id = t.id
	) AS artists,
	t.album,
	t.spotify_url,
	(
		SELECT GROUP_CONCAT(g.name ORDER BY g.name SEPARATOR ', ')
		FROM track_genres tg
		JOIN genres g ON g.id = tg.genre_id
		WHERE tg.track_id = t.id
	) AS genres
FROM playlist_tracks pt
JOIN playlists p ON p.id = pt.playlist_id
JOIN tracks    t ON t.id = pt.track_id;

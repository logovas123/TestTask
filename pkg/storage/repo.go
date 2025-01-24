package storage

import (
	"log/slog"

	"SongLibrary/pkg/song"
)

type SongRepo interface {
	AddSongToDB(*slog.Logger, song.Song) error
	GetSongsFromDB(*slog.Logger, song.Song, int, int) ([]song.Song, error)
	DeleteSongByIDFromDB(*slog.Logger, int) (int, error)
	GetTextOfSongFromDB(*slog.Logger, int) (string, error)
	UpdateSongByID(*slog.Logger, song.SongForUpdate, int) (int, error)
	Close()
}

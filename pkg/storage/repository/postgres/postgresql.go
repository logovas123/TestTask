package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"SongLibrary/pkg/song"
	"SongLibrary/pkg/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SongPostgresRepository struct {
	Pool *pgxpool.Pool
}

func NewSongPostgresRepository(pool *pgxpool.Pool) *SongPostgresRepository {
	return &SongPostgresRepository{
		Pool: pool,
	}
}

func NewConnPostgres(logger *slog.Logger) (*pgxpool.Pool, error) {
	coonString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	slog.Info("Please wait...")
	time.Sleep(3 * time.Second) // поднятие базы

	pool, err := pgxpool.New(context.Background(), coonString)
	if err != nil {
		logger.Error("error create connect to db:",
			"error", err,
		)
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		logger.Error("error ping to db:",
			"error", err,
		)
		return nil, err
	}

	logger.Info("connect to db create success")

	return pool, nil
}

func (repo *SongPostgresRepository) AddSongToDB(logger *slog.Logger, s song.Song) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := repo.Pool.Begin(ctx)
	if err != nil {
		logger.Error("error begin tx", "ERROR", err)
		return err
	}
	defer tx.Rollback(ctx)

	var id int
	err = tx.QueryRow(ctx, "select song_id from songs where song_name = $1 and group_name = $2", s.Song, s.Group).Scan(&id)
	if err == nil {
		logger.Error("this song exist")
		return storage.ErrorSongExist
	} else if err != pgx.ErrNoRows {
		logger.Error("error exec SELECT query to db: ", "ERROR", err)
		return err
	}

	_, err = tx.Exec(ctx, "INSERT INTO songs (song_name, group_name, release_date, text_of_song, link) VALUES ($1, $2, $3, $4, $5)",
		s.Song,
		s.Group,
		s.ReleaseDate,
		s.Text,
		s.Link,
	)
	if err != nil {
		logger.Error("error exec INSERT query to db: ", "ERROR", err)
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		logger.Error("error in tx", "ERROR", err)
		return err
	}

	return nil
}

func (repo *SongPostgresRepository) GetSongsFromDB(logger *slog.Logger, s song.Song, limit int, offset int) ([]song.Song, error) {
	query := "SELECT song_id, song_name, group_name, release_date, text_of_song, link FROM songs WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if s.Song != "" {
		query += fmt.Sprintf(" AND song_name ILIKE $%d", argIndex)
		args = append(args, "%"+s.Song+"%")
		argIndex++
	}
	if s.Group != "" {
		query += fmt.Sprintf(" AND group_name ILIKE $%d", argIndex)
		args = append(args, "%"+s.Group+"%")
		argIndex++
	}
	if s.Text != "" {
		query += fmt.Sprintf(" AND text_of_song ILIKE $%d", argIndex)
		args = append(args, "%"+s.Text+"%")
		argIndex++
	}
	if s.ReleaseDate != "" {
		query += fmt.Sprintf(" AND release_date ILIKE $%d", argIndex)
		args = append(args, "%"+s.ReleaseDate+"%")
		argIndex++
	}
	if s.Link != "" {
		query += fmt.Sprintf(" AND group_name ILIKE $%d", argIndex)
		args = append(args, "%"+s.Link+"%")
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY song_id LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	logger.Debug("result query to db", "query", query)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := repo.Pool.Query(ctx, query, args...)
	if err != nil {
		logger.Error("error exec SELECT query to db: ", "ERROR", err)
		return nil, err
	}
	defer rows.Close()

	songs := []song.Song{}
	for rows.Next() {
		s := song.Song{}
		if err := rows.Scan(&s.SongID, &s.Song, &s.Group, &s.ReleaseDate, &s.Text, &s.Link); err != nil {
			logger.Error("error scan row", "ERROR", err)
			return nil, err
		}
		songs = append(songs, s)
	}

	if len(songs) == 0 {
		logger.Error("error list of songs", "ERROR", storage.ErrorListOfSongsEmpty)
		return nil, storage.ErrorListOfSongsEmpty
	}

	logger.Info("list of songs create success")

	return songs, nil
}

func (repo *SongPostgresRepository) DeleteSongByIDFromDB(logger *slog.Logger, id int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := repo.Pool.Begin(ctx)
	if err != nil {
		logger.Error("error begin tx", "ERROR", err)
		return 0, err
	}
	defer tx.Rollback(ctx)

	var flagID int
	err = tx.QueryRow(ctx, "select song_id from songs where song_id = $1", id).Scan(&flagID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error("song not exist", "id", id)
			return id, storage.ErrorSongNotExist
		}
		logger.Error("error exec SELECT query to db: ", "ERROR", err)
		return 0, err
	}

	_, err = tx.Exec(ctx, "DELETE FROM songs WHERE song_id = $1", id)
	if err != nil {
		logger.Error("error exec DELETE query to db", "ERROR", err)
		return 0, err
	}

	if err = tx.Commit(ctx); err != nil {
		logger.Error("error in tx", "ERROR", err)
		return 0, err
	}

	logger.Info("delete song success", "id", id)
	return id, nil
}

func (repo *SongPostgresRepository) GetTextOfSongFromDB(logger *slog.Logger, id int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var text string
	err := repo.Pool.QueryRow(ctx, "select text_of_song from songs where song_id = $1", id).Scan(&text)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error("song not exist", "id", id)
			return "", storage.ErrorSongNotExist
		}
		logger.Error("error exec SELECT query to db: ", "ERROR", err)
		return "", err
	}

	logger.Info("get text of song success", "id", id)
	return text, nil
}

func (repo *SongPostgresRepository) UpdateSongByID(logger *slog.Logger, s song.SongForUpdate, id int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := repo.Pool.Begin(ctx)
	if err != nil {
		logger.Error("error begin tx", "ERROR", err)
		return 0, err
	}
	defer tx.Rollback(ctx)

	var flagID int
	err = tx.QueryRow(ctx, "select song_id from songs where song_id = $1", id).Scan(&flagID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error("song not exist", "id", id)
			return id, storage.ErrorSongNotExist
		}
		logger.Error("error exec SELECT query to db: ", "ERROR", err)
		return 0, err
	}

	if s.Song == nil && s.Group == nil && s.Text == nil && s.ReleaseDate == nil && s.Link == nil {
		logger.Error("no fields to update")
		return 0, fmt.Errorf("no fields to update")
	}

	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if s.Song != nil {
		updates = append(updates, fmt.Sprintf("song_name = $%d", argIndex))
		args = append(args, *s.Song)
		argIndex++
	}
	if s.Group != nil {
		updates = append(updates, fmt.Sprintf("group_name = $%d", argIndex))
		args = append(args, *s.Group)
		argIndex++
	}
	if s.Text != nil {
		updates = append(updates, fmt.Sprintf("text_of_song = $%d", argIndex))
		args = append(args, *s.Text)
		argIndex++
	}
	if s.ReleaseDate != nil {
		updates = append(updates, fmt.Sprintf("release_date = $%d", argIndex))
		args = append(args, *s.ReleaseDate)
		argIndex++
	}
	if s.Link != nil {
		updates = append(updates, fmt.Sprintf("link = $%d", argIndex))
		args = append(args, *s.Link)
		argIndex++
	}

	args = append(args, id)

	query := fmt.Sprintf("UPDATE songs SET %s WHERE song_id = $%d", strings.Join(updates, ", "), argIndex)
	logger.Debug("get result query", "query", query)

	_, err = tx.Exec(context.Background(), query, args...)
	if err != nil {
		logger.Error("error exec UPDATE query to db", "ERROR", err)
		return 0, err
	}

	if err = tx.Commit(ctx); err != nil {
		logger.Error("error in tx", "ERROR", err)
		return 0, err
	}

	logger.Info("update song success", "id", id)
	return id, nil
}

func (p *SongPostgresRepository) Close() {
	p.Pool.Close()
}

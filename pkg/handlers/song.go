package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"SongLibrary/pkg/song"
	"SongLibrary/pkg/storage"

	"github.com/gorilla/mux"
)

const ApplicationJSON = "application/json"

type SongHandler struct {
	Logger   *slog.Logger
	SongRepo storage.SongRepo
}

// @Summary Add New Song
// @Description Добавляет песню в базу. Принимает json с именем группы и песни
// @Tags songs
// @Accept json
// @Produce json
// @Param song body song.PayloadSong true "Song Information"
// @Success 200 {string} string "Add new song success"
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Internal server error"
// @Router /api/songs [post]
func (h *SongHandler) AddNewSong(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.With(
		"request_id", r.Context().Value("requestID"),
		"url", r.URL.Path,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	if r.Header.Get("Content-Type") != ApplicationJSON {
		http.Error(w, ErrContentType, http.StatusBadRequest)
		logger.Error("Error of Content-Type",
			"ERROR", ErrContentType,
		)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ErrParseBody, http.StatusBadRequest)
		logger.Error("Error from ReadAll",
			"ERROR", err,
		)
		return
	}
	defer r.Body.Close()

	payload := song.PayloadSong{}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		http.Error(w, ErrUnmarshal, http.StatusBadRequest)
		logger.Error("Error of decode json",
			"ERROR", err,
		)
		return
	}
	if payload.Group == "" || payload.Song == "" {
		http.Error(w, ErrFieldEmpty, http.StatusBadRequest)
		logger.Error("Error field of struct",
			"ERROR", ErrFieldEmpty,
			"url", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)
		return
	}

	client := http.Client{}
	host := os.Getenv("EXTERNAL_SERVICE_HOST")
	port := os.Getenv("EXTERNAL_SERVICE_PORT")
	addr := host + ":" + port

	path := "info"

	params := url.Values{}
	params.Add("song", payload.Song)
	params.Add("group", payload.Group)

	baseURL := url.URL{
		Scheme: "http",
		Host:   addr,
		Path:   path,
	}
	baseURL.RawQuery = params.Encode()
	logger.Debug("Create new url:", "url", baseURL.String())

	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		http.Error(w, ErrInternal, http.StatusInternalServerError)
		logger.Error("Error create request",
			"ERROR", err,
		)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, ErrExternalService, http.StatusInternalServerError)
		logger.Error("Error exec request",
			"ERROR", err,
		)
		return
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, ErrExternalService, http.StatusInternalServerError)
		logger.Error("Error read body from external service",
			"ERROR", err,
		)
		return
	}
	defer resp.Body.Close()

	respAPI := song.ResponseFromExternalAPI{}
	err = json.Unmarshal(body, &respAPI)
	if err != nil {
		http.Error(w, ErrExternalService, http.StatusInternalServerError)
		logger.Error("Error decode json from external service",
			"ERROR", err,
		)
		return
	}

	resultSong := song.Song{
		Song:        payload.Song,
		Group:       payload.Group,
		ReleaseDate: respAPI.ReleaseDate,
		Text:        respAPI.Text,
		Link:        respAPI.Link,
	}
	logger.Debug("get result song", "song", fmt.Sprintf("%#v", resultSong))

	err = h.SongRepo.AddSongToDB(logger, resultSong)
	if err != nil {
		if errors.Is(err, storage.ErrorSongExist) {
			http.Error(w, ErrSongExist, http.StatusBadRequest)
			logger.Error("Error add song to db",
				"ERROR", err,
			)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Error("Error add song to db",
			"ERROR", err,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Add new song success"))
	logger.Info("add new song")
}

// @Summary Get a list of songs
// @Description Возвращает список песен с пагинацией и фильтрацией. Принимает query параметры.
// @Tags songs
// @Produce  json
// @Param name query string false "Song Name"
// @Param group query string false "Group Name"
// @Param date query string false "Release Date"
// @Param text query string false "Text"
// @Param link query string false "Link"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Success 200 {array} song.Song "List of songs"
// @Failure 400 {string} string "Bad request"
// @Failure 404 {string} string "No songs found"
// @Failure 500 {string} string "Internal server error"
// @Router /api/songs [get]
func (h *SongHandler) GetListOfSongs(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.With(
		"request_id", r.Context().Value("requestID"),
		"url", r.URL.Path,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	query := r.URL.Query()
	s := song.Song{
		Song:        query.Get("name"),
		Group:       query.Get("group"),
		ReleaseDate: query.Get("date"),
		Text:        query.Get("text"),
		Link:        query.Get("link"),
	}

	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page := 1
	limit := 10
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil {
			http.Error(w, ErrParseQuery, http.StatusBadRequest)
			logger.Error("Error in Atoi",
				"ERROR", err,
			)
			return
		}
		page = p
	}
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, ErrParseQuery, http.StatusBadRequest)
			logger.Error("Error in Atoi",
				"ERROR", err,
			)
			return
		}
		limit = l
	}

	offset := (page - 1) * limit

	songs, err := h.SongRepo.GetSongsFromDB(logger, s, limit, offset)
	if err != nil {
		logger.Error("Error get songs from db",
			"ERROR", err,
		)
		if errors.Is(err, storage.ErrorListOfSongsEmpty) {
			http.Error(w, ErrSongsNotFound, http.StatusNotFound)
			return
		}
		http.Error(w, ErrInternal, http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(songs)
	if err != nil {
		logger.Error("Error marshal list songs",
			"ERROR", err,
		)
		http.Error(w, ErrInternal, http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
	logger.Info("Get list of songs success")
}

// @Summary Delete a song by ID from the library
// @Description Удаляет песню по id
// @Tags song
// @Produce  json
// @Param SONG_ID path int true "Song ID"
// @Success 200 {string} string "Song deleted successfully"
// @Failure 400 {string} string "Bad request"
// @Failure 404 {string} string "Song not found"
// @Failure 500 {string} string "Internal server error"
// @Router /api/song/{SONG_ID} [delete]
func (h *SongHandler) DeleteSongByID(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.With(
		"request_id", r.Context().Value("requestID"),
		"url", r.URL.Path,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	vars := mux.Vars(r)
	idStr := vars["SONG_ID"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, ErrParseQuery, http.StatusBadRequest)
		logger.Error("Error in Atoi",
			"ERROR", err,
		)
		return
	}

	id, err = h.SongRepo.DeleteSongByIDFromDB(logger, id)
	if err != nil {
		logger.Error("Error delete song from db",
			"ERROR", err,
			"id", id,
		)
		if errors.Is(err, storage.ErrorSongNotExist) {
			http.Error(w, ErrSongByIDNotFound, http.StatusNotFound)
			return
		}
		http.Error(w, ErrInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("delete song by id success, id: %v", id)))
	logger.Info("delete song success", "id", id)
}

// @Summary Get the text of song by ID
// @Description Метод возвращает текст песни по id с пагинацией по куплетам. Принимает query параметры.
// @Tags song
// @Produce  json
// @Param SONG_ID path int true "Song ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Limit per page" default(2)
// @Success 200 {array} string "Array of song verses"
// @Failure 400 {string} string "Bad request"
// @Failure 404 {string} string "Song not found or invalid verse range"
// @Failure 500 {string} string "Internal server error"
// @Router /api/song/{SONG_ID} [get]
func (h *SongHandler) GetTextOfSong(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.With(
		"request_id", r.Context().Value("requestID"),
		"url", r.URL.Path,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	vars := mux.Vars(r)
	idStr := vars["SONG_ID"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, ErrParseQuery, http.StatusBadRequest)
		logger.Error("Error in Atoi",
			"ERROR", err,
		)
		return
	}

	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page := 1
	limit := 2
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil {
			http.Error(w, ErrParseQuery, http.StatusBadRequest)
			logger.Error("Error in Atoi",
				"ERROR", err,
			)
			return
		}
		page = p
	}
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, ErrParseQuery, http.StatusBadRequest)
			logger.Error("Error in Atoi",
				"ERROR", err,
			)
			return
		}
		limit = l
	}

	offset := (page - 1) * limit
	text, err := h.SongRepo.GetTextOfSongFromDB(logger, id)
	if err != nil {
		logger.Error("Error delete song from db",
			"ERROR", err,
			"id", id,
		)
		if errors.Is(err, storage.ErrorSongNotExist) {
			http.Error(w, ErrSongByIDNotFound, http.StatusNotFound)
			return
		}
		http.Error(w, ErrInternal, http.StatusInternalServerError)
		return
	}

	verses := strings.Split(text, "\n\n")
	totalVerses := len(verses)

	start := offset
	end := offset + limit
	if start >= totalVerses {
		http.Error(w, "Error amount of verses", http.StatusNotFound)
		logger.Error("Error amount of verses",
			"ERROR", err,
			"id", id,
		)
		return
	}
	if end > totalVerses {
		end = totalVerses
	}

	body, err := json.Marshal(verses[start:end])
	if err != nil {
		logger.Error("Error marshal text of songs",
			"ERROR", err,
		)
		http.Error(w, ErrInternal, http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
	logger.Info("get text of song by id success", "id", id)
}

// @Summary Update to song by id
// @Description Обновляет поля по id. Принимает json, который содержит поля для обновления.
// @Tags song
// @Accept json
// @Produce json
// @Param SONG_ID path int true "ID of song"
// @Param song body song.SongForUpdate true "Data for update"
// @Success 200 {string} string "update song by id success"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Song not found"
// @Failure 500 {string} string "Internal server error"
// @Router /api/song/{SONG_ID} [put]
func (h *SongHandler) UpdateSong(w http.ResponseWriter, r *http.Request) {
	logger := h.Logger.With(
		"request_id", r.Context().Value("requestID"),
		"url", r.URL.Path,
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	vars := mux.Vars(r)
	idStr := vars["SONG_ID"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, ErrParseQuery, http.StatusBadRequest)
		logger.Error("Error in Atoi",
			"ERROR", err,
		)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ErrParseBody, http.StatusBadRequest)
		logger.Error("Error from ReadAll",
			"ERROR", err,
		)
		return
	}
	defer r.Body.Close()

	payload := song.SongForUpdate{}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		http.Error(w, ErrUnmarshal, http.StatusBadRequest)
		logger.Error("Error of decode json",
			"ERROR", err,
		)
		return
	}

	id, err = h.SongRepo.UpdateSongByID(logger, payload, id)
	if err != nil {
		logger.Error("Error update song in db",
			"ERROR", err,
			"id", id,
		)
		if errors.Is(err, storage.ErrorSongNotExist) {
			http.Error(w, ErrSongByIDNotFound, http.StatusNotFound)
			return
		}
		http.Error(w, ErrInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("update song by id success, id: %v", id)))
	logger.Info("delete song success", "id", id)
}

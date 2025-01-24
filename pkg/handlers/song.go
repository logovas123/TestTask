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

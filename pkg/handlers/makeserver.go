package handlers

import (
	"log/slog"
	"net/http"

	"SongLibrary/pkg/middleware"

	"github.com/gorilla/mux"
)

func NewMuxServer(songHandler *SongHandler, logger *slog.Logger) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/api/songs", songHandler.GetListOfSongs).Methods(http.MethodGet)
	r.HandleFunc("/api/songs", songHandler.AddNewSong).Methods(http.MethodPost)
	r.HandleFunc("/api/song/{SONG_ID}", songHandler.DeleteSongByID).Methods(http.MethodDelete)
	r.HandleFunc("/api/song/{SONG_ID}", songHandler.UpdateSong).Methods(http.MethodPut)
	r.HandleFunc("/api/song/{SONG_ID}", songHandler.GetTextOfSong).Methods(http.MethodGet)

	mux := middleware.AccessLog(songHandler.Logger, r)
	mux = middleware.Panic(mux)

	return mux
}

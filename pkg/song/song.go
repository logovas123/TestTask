package song

type Song struct {
	SongID      int64  `json:"song_id"`
	Song        string `json:"song"`
	Group       string `json:"group"`
	ReleaseDate string `json:"releaseDate"`
	Text        string `json:"text"`
	Link        string `json:"link"`
}

type PayloadSong struct {
	Song  string `json:"song"`
	Group string `json:"group"`
}

type ResponseFromExternalAPI struct {
	ReleaseDate string `json:"releaseDate"`
	Text        string `json:"text"`
	Link        string `json:"link"`
}

type SongForUpdate struct {
	Song        *string `json:"song"`
	Group       *string `json:"group"`
	ReleaseDate *string `json:"releaseDate"`
	Text        *string `json:"text"`
	Link        *string `json:"link"`
}

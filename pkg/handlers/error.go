package handlers

var (
	ErrContentType      = "bad Content-Type"
	ErrParseBody        = "cant read body request"
	ErrUnmarshal        = "cant decode body"
	ErrFieldEmpty       = "field of struct payload empty"
	ErrInternal         = "internal error"
	ErrExternalService  = "error from external service"
	ErrParseQuery       = "error parse value of query param"
	ErrSongsNotFound    = "songs not found"
	ErrSongByIDNotFound = "song by id not found"
	ErrSongExist        = "song exist"
)

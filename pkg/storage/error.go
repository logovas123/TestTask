package storage

import "fmt"

// кастомные ошибки
var (
	ErrorSongExist        = fmt.Errorf("song with this ID exist")
	ErrorSongNotExist     = fmt.Errorf("song not exist")
	ErrorListOfSongsEmpty = fmt.Errorf("list of songs empty")
)

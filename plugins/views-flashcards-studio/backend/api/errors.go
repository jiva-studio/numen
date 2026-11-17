package api

// ErrorCode names why a read or a write of a note failed. It is carried in the `error` field of a response; an empty value is no error.
type ErrorCode string

// ErrorCode values.
const (
	ErrorCodeUnspecified    ErrorCode = ""
	ErrorCodeNotFound       ErrorCode = "not_found"
	ErrorCodeNotANote       ErrorCode = "not_a_note"
	ErrorCodeNotText        ErrorCode = "not_text"
	ErrorCodeTooLarge       ErrorCode = "too_large"
	ErrorCodeBodyUnwritable ErrorCode = "body_unwritable"
	ErrorCodeUnreadable     ErrorCode = "unreadable"
	ErrorCodeAlreadyExists  ErrorCode = "already_exists"
	ErrorCodeInvalidName    ErrorCode = "invalid_name"
	ErrorCodeNotACardModel  ErrorCode = "not_a_card_model"
	ErrorCodeNotADeck       ErrorCode = "not_a_deck"
	ErrorCodeDeckTooLarge   ErrorCode = "deck_too_large"
	ErrorCodeNotAPreset     ErrorCode = "not_a_preset"
	ErrorCodeConflict       ErrorCode = "conflict"
)

// FileStat identifies a version of a file: its path, its size in bytes, its modification time in
// nanoseconds since the epoch, the SHA-256 of its content in lowercase hex and the number of writes the
// vault had taken when the answer was made. Mtime and Version are encoded as decimal strings because a
// nanosecond count and a 64-bit counter are larger than the integers a JSON number carries without loss.
type FileStat struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mtime   int64  `json:"mtime,string"`
	Hash    string `json:"hash"`
	Version uint64 `json:"version,string"`
}

// Bounds is the closed range a number may take.
type Bounds struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

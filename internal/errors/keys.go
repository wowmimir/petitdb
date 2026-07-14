package errors

import "errors"

var (
	ErrEmptyKey    = errors.New("ERR key cannot be empty")
	ErrKeyTooLong  = errors.New("ERR key too long (max 256 characters)")
)
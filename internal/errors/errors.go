package errors

import "errors"

var (
    ErrEmptyKey   = errors.New("ERR key cannot be empty")
    ErrKeyTooLong = errors.New("ERR key length exceeds 256 characters")
    // We'll handle wrong args and unknown commands in the dispatcher itself.
)
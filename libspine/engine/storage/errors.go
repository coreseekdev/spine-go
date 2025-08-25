package storage

import "errors"

var (
	ErrWrongType  = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	ErrNotInteger = errors.New("ERR value is not an integer or out of range")
	ErrNotFloat   = errors.New("ERR value is not a valid float")
	ErrIndexOutOfRange = errors.New("ERR index out of range")
	ErrNoSuchKey  = errors.New("ERR no such key")
)

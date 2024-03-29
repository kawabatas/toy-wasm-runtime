package wasm

import "errors"

var (
	ErrInvalidByte        = errors.New("invalid byte")
	ErrInvalidMagicNumber = errors.New("invalid magic number")
	ErrInvalidVersion     = errors.New("invalid version header")
	ErrInvalidSectionID   = errors.New("invalid section id")
)

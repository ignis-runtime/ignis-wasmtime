package models

//go:generate stringer -type=RuntimeType -trimprefix=RuntimeType -output=runtime_type_string.go
type RuntimeType int

const (
	RuntimeTypeWASM RuntimeType = iota
	RuntimeTypeJS
)

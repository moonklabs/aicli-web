package ansi

import "io"

// ANSIParserInterface defines the public interface for ANSI parsing
type ANSIParserInterface interface {
	Parse(data []byte) ([]ANSICommand, error)
	ParseStream(reader io.Reader) (<-chan ANSICommand, error)
	Reset() error
	GetState() ParserState
	GetStatistics() *ParserStatistics
}
package core

import (
	"fmt"
	"os"
)

// SaveFile interface represents a loaded game save across any generation
type SaveFile interface {
	Generation() int
	GetParty() ([]Pokemon, error)
	InjectParty(party []Pokemon) error
	WriteToFile(path string) error
	WriteToBytes() []byte
}

// SaveFactory registers parsing functions for different generations
var parsers []func([]byte) (SaveFile, error)

// tradeReady tracks generations whose party writer has been verified to
// round-trip a save byte-for-byte. A generation is only allowed to trade once
// it registers here; everything else is refused rather than silently no-op'd.
var tradeReady = map[int]bool{}

// RegisterTradeReady marks a generation as having a complete, verified writer.
func RegisterTradeReady(gen int) {
	tradeReady[gen] = true
}

// SupportsTrade reports whether a generation can participate in a trade.
func SupportsTrade(gen int) bool {
	return tradeReady[gen]
}

func RegisterSaveParser(parser func([]byte) (SaveFile, error)) {
	parsers = append(parsers, parser)
}

// LoadSave bytes attempts to identify and load a save file from raw bytes
func LoadSaveFromBytes(data []byte) (SaveFile, error) {
	for _, parser := range parsers {
		save, err := parser(data)
		if err == nil && save != nil {
			return save, nil
		}
	}
	return nil, fmt.Errorf("unsupported save file format or corrupted data")
}

// LoadSave reads a file from disk and attempts to parse it
func LoadSave(path string) (SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadSaveFromBytes(data)
}

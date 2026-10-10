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

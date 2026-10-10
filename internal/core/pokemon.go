package core

// Pokemon represents the common interface for a Pokemon across all generations.
// A generation-specific struct (e.g., gen3.Pokemon) will implement this.
type Pokemon interface {
	GetSpecies() uint16
	GetNickname() string
	GetLevel() byte
	GetMoves() [4]uint16
	GetHeldItem() uint16
	GetTrainerName() string
	GetTrainerID() uint32 // Combined TID/SID where applicable

	// Core Engine
	GetGeneration() int
	ToBytes() []byte
	IsChecksumValid() bool
}

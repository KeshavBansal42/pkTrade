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

// NationalSpecies is implemented by records that expose the National Dex
// number, which is the identity used across generations.
type NationalSpecies interface {
	NationalID() uint16
}

// CaptureBall is implemented by records that store the ball a Pokémon was
// caught in.
type CaptureBall interface {
	GetCaptureBall() (byte, bool)
}

// NationalSpeciesID returns the National Dex number of a record, falling back
// to its stored species id when the record does not implement NationalSpecies.
func NationalSpeciesID(p Pokemon) uint16 {
	if species, ok := p.(NationalSpecies); ok {
		return species.NationalID()
	}
	return p.GetSpecies()
}

// KnownCaptureBall reports whether a record's capture ball is known.
func KnownCaptureBall(p Pokemon) bool {
	ball, ok := p.(CaptureBall)
	if !ok {
		return false
	}
	_, known := ball.GetCaptureBall()
	return known
}

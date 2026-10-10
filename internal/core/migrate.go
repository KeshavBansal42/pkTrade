package core

import "fmt"

// Migrate converts a specific generation's Pokemon into a UniversalPokemon
func MigrateToUPF(p Pokemon) *UniversalPokemon {
	upf := &UniversalPokemon{
		Species:     p.GetSpecies(),
		Nickname:    p.GetNickname(),
		Level:       p.GetLevel(),
		Moves:       p.GetMoves(),
		HeldItem:    p.GetHeldItem(),
		TrainerName: p.GetTrainerName(),
	}

	// Handle generation-specific UPF derivations
	switch p.GetGeneration() {
	case 1:
		// Map Gen 1 DVs to modern IVs: IV = (DV * 2) + 1
		// Gen 1 doesn't have an OTID split or Abilities.
		upf.HiddenAbility = true // Virtual Console rule
		upf.Nature = 0           // Real logic requires exp % 25
	case 3:
		// Gen 3 native IVs and PID
		upf.TID = uint16(p.GetTrainerID() & 0xFFFF)
		upf.SID = uint16(p.GetTrainerID() >> 16)
	case 4, 5, 6, 7, 8:
		upf.TID = uint16(p.GetTrainerID() & 0xFFFF)
		upf.SID = uint16(p.GetTrainerID() >> 16)
	}

	return upf
}

// MigrateFromUPF converts a UPF back into a specific generation's Pokemon format
func MigrateFromUPF(upf *UniversalPokemon, targetGen int) (Pokemon, error) {
	switch targetGen {
	case 1:
		return nil, fmt.Errorf("migrating backwards to Gen 1 is technically restricted/lossy")
	case 3:
		// Convert UPF to Gen 3 struct (Requires generating block shuffles and encryption keys)
		return nil, fmt.Errorf("gen 3 writer not yet implemented")
	case 8:
		// Convert UPF to Gen 8 344-byte flat array
		return nil, fmt.Errorf("gen 8 writer not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported target generation: %d", targetGen)
	}
}

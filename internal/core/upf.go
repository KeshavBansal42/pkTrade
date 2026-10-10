package core

// UniversalPokemon is the UPF (Universal Pokemon Format).
// It acts as an intermediate state holding a superset of all possible data fields
// across all 8 generations.
type UniversalPokemon struct {
	// Core Identity
	Species uint16
	Form    byte
	PID     uint32
	Nickname string

	// Stats & Growth
	IVs    [6]byte // 0-31 for Gen 3+, derived for Gen 1/2. Order: HP, Atk, Def, SpA, SpD, Spe
	EVs    [6]byte // 0-252 for Gen 3+, Stat Exp for Gen 1/2 mapped to EVs
	Nature byte
	Level  byte
	Experience uint32

	// Moves
	Moves [4]uint16
	PP    [4]byte
	PPUps [4]byte

	// Held Item
	HeldItem uint16

	// Trainer Info
	TrainerName string
	TID         uint16
	SID         uint16
	Gender      byte

	// Modern Features (Null/0 for older gens)
	Ability       uint16
	HiddenAbility bool
	HyperTrained  [6]bool
	DynamaxLevel  byte
	Gigantamax    bool
	MintNature    byte // Gen 8 stat overrides

	// Origin
	OriginLanguage byte
	OriginGame     byte
	Markings       uint16
}

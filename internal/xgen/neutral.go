// Package xgen converts Pokémon records between the supported generations
// (1 to 8) through one generation-independent representation. Every conversion
// goes native -> Neutral -> native. A conversion never mutates its input; the
// only side effect is the loss list it returns, which names the source data the
// destination format cannot represent.
package xgen

import "errors"

// Slot says which native record size to build on import.
type Slot int

const (
	Box Slot = iota
	Party
)

// Neutral is the generation-independent form of one record. Every field is in
// a canonical unit; the zero value means "not present in the source" and is
// guarded by the matching Has* flag where that matters.
type Neutral struct {
	// Identity.
	Species uint16 // National Dex id, 1..898. 0 is a hard block.
	Form    uint8  // Form index; only 0 (base) is written today.
	Nick    string
	OT      string
	TID     uint16
	SID     uint16
	HasSID  bool
	IsEgg   bool

	// Growth.
	Exp       uint32
	Level     byte
	Nature    byte
	HasNature bool

	// Genetics.
	PID         uint32
	HasPID      bool
	Gender      byte // 0 male, 1 female, 2 genderless
	IVs         [6]byte
	EVs         [6]uint16
	AbilitySlot byte
	Shiny       bool

	// Battle.
	Moves    [4]uint16
	MovePP   [4]byte
	MovePPUp [4]byte

	// Items and origin.
	HeldItem uint16 // canonical (Gen 4+) item id
	Ball     byte   // canonical (Gen 4+) ball id
	HasBall  bool

	// Optional per-generation data.
	Friendship    byte
	HasFriendship bool
	Pokerus       byte
	HasPokerus    bool
	MetLevel      byte
	MetLocation   byte
	HasMet        bool

	// Party display data; the real stats are always recomputed on import.
	CurrentHP uint16

	SourceGen   int
	SourceParty bool
}

// Loss is one piece of source data the destination format cannot represent.
type Loss struct {
	Field  string
	Reason string
}

// Losses is the ordered list of losses for one direction of a conversion.
type Losses []Loss

// Strings renders the losses for the trade preview.
func (l Losses) Strings() []string {
	out := make([]string, 0, len(l))
	for _, loss := range l {
		if loss.Field == "" {
			out = append(out, loss.Reason)
			continue
		}
		out = append(out, loss.Field+": "+loss.Reason)
	}
	return out
}

func (l *Losses) add(field, reason string) {
	*l = append(*l, Loss{Field: field, Reason: reason})
}

// Hard blocks. A conversion that hits one of these must be refused.
var (
	ErrEgg           = errors.New("eggs cannot be traded")
	ErrNoSpecies     = errors.New("the record has no species")
	ErrSpeciesAbsent = errors.New("the species does not exist in the destination generation")
	ErrInvalidRecord = errors.New("the record is not valid")
)

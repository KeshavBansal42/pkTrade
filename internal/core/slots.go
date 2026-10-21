package core

import (
	"fmt"
	"strconv"
	"strings"
)

// SlotKind identifies which part of a save a Pokémon slot lives in.
type SlotKind string

const (
	PartySlot SlotKind = "party"
	BoxSlot   SlotKind = "box"
)

// SlotRef is a one-based reference to a party or box slot. Every generation
// uses the same syntax in the CLI and mobile API: party:N or box:B/S.
type SlotRef struct {
	Kind  SlotKind `json:"kind"`
	Party int      `json:"party,omitempty"`
	Box   int      `json:"box,omitempty"`
	Slot  int      `json:"slot,omitempty"`
}

func Party(n int) SlotRef { return SlotRef{Kind: PartySlot, Party: n} }
func Box(b, s int) SlotRef {
	return SlotRef{Kind: BoxSlot, Box: b, Slot: s}
}

func (s SlotRef) String() string {
	if s.Kind == PartySlot {
		return fmt.Sprintf("party:%d", s.Party)
	}
	return fmt.Sprintf("box:%d/%d", s.Box, s.Slot)
}

// Validate checks the syntax of a slot reference without knowing the save's
// capacities. Capacity checks are done by the trade engine.
func (s SlotRef) Validate() error {
	switch s.Kind {
	case PartySlot:
		if s.Party < 1 {
			return fmt.Errorf("party slot must be at least 1")
		}
	case BoxSlot:
		if s.Box < 1 {
			return fmt.Errorf("box number must be at least 1")
		}
		if s.Slot < 1 {
			return fmt.Errorf("box slot must be at least 1")
		}
	default:
		return fmt.Errorf("unknown slot kind %q (use party or box)", s.Kind)
	}
	return nil
}

// ParseSlot parses party:N and box:B/S. The box forms box:B and box:B/S are
// both accepted; box:B means slot 1.
func ParseSlot(value string) (SlotRef, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	switch {
	case strings.HasPrefix(value, "party:"):
		n, err := strconv.Atoi(strings.TrimPrefix(value, "party:"))
		if err != nil {
			return SlotRef{}, fmt.Errorf("invalid party slot %q", value)
		}
		return Party(n), nil
	case strings.HasPrefix(value, "box:"):
		rest := strings.TrimPrefix(value, "box:")
		parts := strings.Split(rest, "/")
		if len(parts) > 2 {
			return SlotRef{}, fmt.Errorf("invalid box slot %q (use box:B/S)", value)
		}
		b, err := strconv.Atoi(parts[0])
		if err != nil {
			return SlotRef{}, fmt.Errorf("invalid box number %q", parts[0])
		}
		slot := 1
		if len(parts) == 2 {
			slot, err = strconv.Atoi(parts[1])
			if err != nil {
				return SlotRef{}, fmt.Errorf("invalid box slot %q", parts[1])
			}
		}
		return Box(b, slot), nil
	default:
		return SlotRef{}, fmt.Errorf("invalid slot %q (use party:N or box:B/S)", value)
	}
}

// GameInfo describes the identified save. Family is the save-format family
// (for example "frlg", "rse", "dp", "hgss"), not just the generation: records
// from different families within one generation are not interchangeable.
type GameInfo struct {
	Generation int
	Family     string
	Game       string
	SaveIndex  uint32
	ActiveSlot string
}

// CompatibilityKey is the minimum information two saves must share before any
// record can be exchanged. Generation alone is deliberately insufficient.
type CompatibilityKey struct {
	Generation int
	Family     string
	Format     int
}

// TrainerInfo is the real player identity stored in the save.
type TrainerInfo struct {
	Name   string `json:"name"`
	TID    uint16 `json:"tid"`
	SID    uint16 `json:"sid"`
	Gender byte   `json:"gender,omitempty"`
}

// Capabilities reports which operations a save implementation actually
// supports. The UI and CLI must not claim a feature that is false here.
type Capabilities struct {
	Party     bool `json:"party"`
	Boxes     bool `json:"boxes"`
	Evolution bool `json:"evolution"`
	Dex       bool `json:"dex"`
}

// Evolution describes one evolution performed during a trade.
type Evolution struct {
	From     uint16 `json:"from"`
	To       uint16 `json:"to"`
	Item     uint16 `json:"item,omitempty"`
	ItemName string `json:"itemName,omitempty"`
	FromName string `json:"fromName,omitempty"`
	ToName   string `json:"toName,omitempty"`
}

// SlotEntry is one occupied slot returned by list operations. Empty entries
// are only included when explicitly requested.
type SlotEntry struct {
	Ref     SlotRef `json:"ref"`
	Pokemon Pokemon `json:"-"`
	Empty   bool    `json:"empty"`
}

// InfoProvider is implemented by saves that can describe themselves.
type InfoProvider interface {
	Info() GameInfo
}

// CompatibilityProvider is implemented by saves that can state exactly which
// record format they use. It is required for trading.
type CompatibilityProvider interface {
	CompatibilityKey() CompatibilityKey
}

// TrainerProvider is implemented by saves that can parse the real trainer.
type TrainerProvider interface {
	TrainerInfo() (TrainerInfo, error)
}

// CapabilityProvider exposes the features a save implementation supports.
type CapabilityProvider interface {
	Capabilities() Capabilities
}

// Validator is implemented by saves that can validate their own integrity.
type Validator interface {
	Validate() error
}

// Storage is the slot-level interface used by the shared trade engine.
// Implementations must convert records to the destination slot's on-disk
// format inside WriteSlot and must not mutate their input Pokemon.
type Storage interface {
	PartyCapacity() int
	BoxCount() int
	BoxCapacity() int
	ReadSlot(SlotRef) (Pokemon, error)
	WriteSlot(SlotRef, Pokemon) error
	ListSlots(includeEmpty bool) ([]SlotEntry, error)
}

// TradeHooks lets a generation apply its own rules to a received record. The
// received record is in the source save's format. Hooks must return a new
// value and must not modify the input.
type TradeHooks interface {
	FixupReceived(Pokemon) (Pokemon, []Evolution, error)
	MarkDex(Pokemon) error
}

// TradeLegality lets a Pokemon reject itself from a trade, for example eggs,
// bad eggs, or records that failed a deeper validity check.
type TradeLegality interface {
	TradeBlockReason() string
}

// OutputVerifier is an optional stronger integrity check. It receives the
// original save bytes and the bytes that would be written, and must confirm
// that only the expected regions changed.
type OutputVerifier interface {
	VerifyOutput(original, written []byte) error
}

// partyStorage adapts the original party-only SaveFile interface for
// generations that do not yet implement Storage.
type partyStorage struct {
	save SaveFile
}

func (p partyStorage) PartyCapacity() int { return 6 }
func (p partyStorage) BoxCount() int      { return 0 }
func (p partyStorage) BoxCapacity() int   { return 0 }

func (p partyStorage) ReadSlot(ref SlotRef) (Pokemon, error) {
	if ref.Kind != PartySlot {
		return nil, fmt.Errorf("this save has no box support")
	}
	party, err := p.save.GetParty()
	if err != nil {
		return nil, err
	}
	if ref.Party < 1 || ref.Party > len(party) {
		return nil, fmt.Errorf("party slot %d is empty or does not exist", ref.Party)
	}
	return party[ref.Party-1], nil
}

func (p partyStorage) WriteSlot(ref SlotRef, incoming Pokemon) error {
	if ref.Kind != PartySlot {
		return fmt.Errorf("this save has no box support")
	}
	party, err := p.save.GetParty()
	if err != nil {
		return err
	}
	if ref.Party < 1 || ref.Party > len(party) {
		return fmt.Errorf("party slot %d does not exist", ref.Party)
	}
	party[ref.Party-1] = incoming
	return p.save.InjectParty(party)
}

func (p partyStorage) ListSlots(includeEmpty bool) ([]SlotEntry, error) {
	party, err := p.save.GetParty()
	if err != nil {
		return nil, err
	}
	out := make([]SlotEntry, 0, 6)
	for i := 0; i < 6; i++ {
		ref := Party(i + 1)
		if i < len(party) {
			out = append(out, SlotEntry{Ref: ref, Pokemon: party[i]})
		} else if includeEmpty {
			out = append(out, SlotEntry{Ref: ref, Empty: true})
		}
	}
	return out, nil
}

func storageFor(save SaveFile) Storage {
	if s, ok := save.(Storage); ok {
		return s
	}
	return partyStorage{save: save}
}

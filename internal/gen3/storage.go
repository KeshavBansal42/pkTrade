package gen3

import (
	"fmt"

	"pktrade/internal/core"
)

const (
	BoxCount    = 14
	BoxCapacity = 30
	BoxRecord   = 80
	pcBoxStart  = 4
)

type slotStorage struct {
	save *SaveFile
}

func (s *SaveFile) PartyCapacity() int { return 6 }
func (s *SaveFile) BoxCount() int      { return BoxCount }
func (s *SaveFile) BoxCapacity() int   { return BoxCapacity }

func (s *SaveFile) ReadSlot(ref core.SlotRef) (core.Pokemon, error) {
	return (&slotStorage{save: s}).ReadSlot(ref)
}

func (s *SaveFile) WriteSlot(ref core.SlotRef, incoming core.Pokemon) error {
	return (&slotStorage{save: s}).WriteSlot(ref, incoming)
}

func (s *SaveFile) ListSlots(includeEmpty bool) ([]core.SlotEntry, error) {
	return (&slotStorage{save: s}).ListSlots(includeEmpty)
}

func (s *slotStorage) ReadSlot(ref core.SlotRef) (core.Pokemon, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if s.save == nil || s.save.ActiveSlot == nil {
		return nil, fmt.Errorf("active save slot is missing")
	}
	switch ref.Kind {
	case core.PartySlot:
		party, err := GetParty(s.save.ActiveSlot)
		if err != nil {
			return nil, err
		}
		if ref.Party < 1 || ref.Party > len(party) {
			return nil, fmt.Errorf("party slot %d is empty or does not exist", ref.Party)
		}
		return party[ref.Party-1], nil
	case core.BoxSlot:
		if ref.Box < 1 || ref.Box > BoxCount || ref.Slot < 1 || ref.Slot > BoxCapacity {
			return nil, fmt.Errorf("box slot %s is outside the 14 boxes of 30 slots", ref)
		}
		buf, err := s.save.pcBuffer()
		if err != nil {
			return nil, err
		}
		off := s.save.boxRecordOffset(ref.Box, ref.Slot)
		if off+BoxRecord > len(buf) {
			return nil, fmt.Errorf("box slot %s is outside the PC buffer", ref)
		}
		record := buf[off : off+BoxRecord]
		if isZero(record) {
			return nil, fmt.Errorf("box slot %s is empty", ref)
		}
		poke, err := ParsePokemon(record)
		if err != nil {
			return nil, err
		}
		if poke.Species == 0 {
			return nil, fmt.Errorf("box slot %s is empty", ref)
		}
		if !poke.ChecksumValid {
			return nil, fmt.Errorf("box slot %s has an invalid checksum", ref)
		}
		poke.BoxNum, poke.BoxSlot = ref.Box, ref.Slot
		return poke, nil
	default:
		return nil, fmt.Errorf("unknown slot kind %q", ref.Kind)
	}
}

func (s *slotStorage) WriteSlot(ref core.SlotRef, incoming core.Pokemon) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	poke, ok := incoming.(*Pokemon)
	if !ok {
		return fmt.Errorf("cannot write a non-Gen 3 record; cross-generation writes are not supported")
	}
	switch ref.Kind {
	case core.PartySlot:
		party, err := GetParty(s.save.ActiveSlot)
		if err != nil {
			return err
		}
		if ref.Party < 1 || ref.Party > 6 {
			return fmt.Errorf("party slot %d is outside the six-slot party", ref.Party)
		}
		data, err := poke.ToPartyData()
		if err != nil {
			return fmt.Errorf("party slot %s: %w", ref, err)
		}
		expanded, err := ParsePokemon(data)
		if err != nil {
			return fmt.Errorf("party slot %s: %w", ref, err)
		}
		for len(party) < ref.Party {
			party = append(party, nil)
		}
		if ref.Party > len(party) {
			return fmt.Errorf("party slot %s does not exist", ref)
		}
		party[ref.Party-1] = expanded
		out := make([]core.Pokemon, 0, len(party))
		for i, member := range party {
			if member == nil {
				return fmt.Errorf("party slot %d is empty; party trades cannot create a gap", i+1)
			}
			out = append(out, member)
		}
		return s.save.InjectParty(out)
	case core.BoxSlot:
		if ref.Box < 1 || ref.Box > BoxCount || ref.Slot < 1 || ref.Slot > BoxCapacity {
			return fmt.Errorf("box slot %s is outside the 14 boxes of 30 slots", ref)
		}
		data, err := poke.ToBoxData()
		if err != nil {
			return fmt.Errorf("box slot %s: %w", ref, err)
		}
		parsed, err := ParsePokemon(data)
		if err != nil {
			return fmt.Errorf("box slot %s: %w", ref, err)
		}
		if !parsed.ChecksumValid || parsed.Species == 0 {
			return fmt.Errorf("box slot %s: record is not a valid stored Pokémon", ref)
		}
		buf, err := s.save.pcBuffer()
		if err != nil {
			return err
		}
		off := s.save.boxRecordOffset(ref.Box, ref.Slot)
		copy(buf[off:off+BoxRecord], data)
		return s.save.writePCBuffer(buf)
	default:
		return fmt.Errorf("unknown slot kind %q", ref.Kind)
	}
}

func (s *slotStorage) ListSlots(includeEmpty bool) ([]core.SlotEntry, error) {
	out := make([]core.SlotEntry, 0, 6+BoxCount*BoxCapacity)
	party, err := GetParty(s.save.ActiveSlot)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 6; i++ {
		ref := core.Party(i + 1)
		if i < len(party) {
			out = append(out, core.SlotEntry{Ref: ref, Pokemon: party[i]})
		} else if includeEmpty {
			out = append(out, core.SlotEntry{Ref: ref, Empty: true})
		}
	}
	buf, err := s.save.pcBuffer()
	if err != nil {
		return nil, err
	}
	for box := 1; box <= BoxCount; box++ {
		for slot := 1; slot <= BoxCapacity; slot++ {
			ref := core.Box(box, slot)
			off := s.save.boxRecordOffset(box, slot)
			record := buf[off : off+BoxRecord]
			if isZero(record) {
				if includeEmpty {
					out = append(out, core.SlotEntry{Ref: ref, Empty: true})
				}
				continue
			}
			poke, err := ParsePokemon(record)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", ref, err)
			}
			if poke.Species == 0 {
				if includeEmpty {
					out = append(out, core.SlotEntry{Ref: ref, Empty: true})
				}
				continue
			}
			if !poke.ChecksumValid {
				return nil, fmt.Errorf("%s has an invalid checksum", ref)
			}
			poke.BoxNum, poke.BoxSlot = box, slot
			out = append(out, core.SlotEntry{Ref: ref, Pokemon: poke})
		}
	}
	return out, nil
}

func (s *slotStorage) boxPokemon() ([]*Pokemon, error) {
	entries, err := s.ListSlots(false)
	if err != nil {
		return nil, err
	}
	out := make([]*Pokemon, 0, len(entries))
	for _, entry := range entries {
		if entry.Ref.Kind != core.BoxSlot {
			continue
		}
		if poke, ok := entry.Pokemon.(*Pokemon); ok {
			out = append(out, poke)
		}
	}
	return out, nil
}

func (s *SaveFile) boxRecordOffset(box, slot int) int {
	return pcBoxStart + ((box-1)*BoxCapacity+(slot-1))*BoxRecord
}

// pcBuffer concatenates the data portions of sections 5 through 13. The first
// four bytes are the current box index; box records start at offset 4. The
// storage area also holds the battle box and the daycare, so the buffer is
// larger than the box records alone.
func (s *SaveFile) pcBuffer() ([]byte, error) {
	if s.ActiveSlot == nil {
		return nil, fmt.Errorf("active save slot is missing")
	}
	var buf []byte
	for id := uint16(5); id <= 13; id++ {
		sec, ok := s.ActiveSlot.Sections[id]
		if !ok || len(sec) != SectionSize {
			return nil, fmt.Errorf("PC section %d is missing or truncated", id)
		}
		buf = append(buf, sec[:sectionDataSizes[id]]...)
	}
	if len(buf) < pcBoxStart+BoxCount*BoxCapacity*BoxRecord {
		return nil, fmt.Errorf("PC buffer is too short: %d bytes", len(buf))
	}
	return buf, nil
}

func (s *SaveFile) writePCBuffer(buf []byte) error {
	if s.ActiveSlot == nil {
		return fmt.Errorf("active save slot is missing")
	}
	if len(buf) < pcBoxStart+BoxCount*BoxCapacity*BoxRecord {
		return fmt.Errorf("PC buffer is too short: %d bytes", len(buf))
	}
	pos := 0
	for id := uint16(5); id <= 13; id++ {
		sec, ok := s.ActiveSlot.Sections[id]
		if !ok || len(sec) != SectionSize {
			return fmt.Errorf("PC section %d is missing or truncated", id)
		}
		size := sectionDataSizes[id]
		copy(sec[:size], buf[pos:pos+size])
		pos += size
	}
	return nil
}

func isZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

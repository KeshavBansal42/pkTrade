package gen1

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/core"
)

const (
	BoxCount    = 12
	BoxCapacity = 20
	boxListSize = 0x462
	currentBox  = 0x30C0
	currentIdx  = 0x284C
)

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

type slotStorage struct{ save *SaveFile }

func boxListOffset(box int) int {
	if box <= 6 {
		return 0x4000 + (box-1)*boxListSize
	}
	return 0x6000 + (box-7)*boxListSize
}

func (s *slotStorage) ReadSlot(ref core.SlotRef) (core.Pokemon, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	switch ref.Kind {
	case core.PartySlot:
		party, err := s.save.GetParty()
		if err != nil {
			return nil, err
		}
		if ref.Party < 1 || ref.Party > len(party) {
			return nil, fmt.Errorf("party slot %d is empty or does not exist", ref.Party)
		}
		return party[ref.Party-1], nil
	case core.BoxSlot:
		if ref.Box < 1 || ref.Box > BoxCount || ref.Slot < 1 || ref.Slot > BoxCapacity {
			return nil, fmt.Errorf("box slot %s is outside the 12 boxes of 20 slots", ref)
		}
		base := boxListOffset(ref.Box)
		if base+boxListSize > len(s.save.data) {
			return nil, fmt.Errorf("box %d is outside the save file", ref.Box)
		}
		list := s.save.data[base : base+boxListSize]
		count := int(list[0])
		if count > BoxCapacity || ref.Slot > count {
			return nil, fmt.Errorf("box slot %s is empty", ref)
		}
		marker := list[1+ref.Slot-1]
		if marker == 0 || marker == 0xFF {
			return nil, fmt.Errorf("box slot %s is empty", ref)
		}
		bodyOff := 22 + (ref.Slot-1)*BoxRecordSize
		otOff := 22 + BoxCapacity*BoxRecordSize + (ref.Slot-1)*NameLength
		nickOff := otOff + BoxCapacity*NameLength
		poke, err := ParsePokemon(list[bodyOff:bodyOff+BoxRecordSize], list[nickOff:nickOff+NameLength], list[otOff:otOff+NameLength])
		if err != nil {
			return nil, err
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
		return fmt.Errorf("cannot write a non-Gen 1 record; cross-generation writes are not supported")
	}
	switch ref.Kind {
	case core.PartySlot:
		party, err := s.save.GetParty()
		if err != nil {
			return err
		}
		if ref.Party < 1 || ref.Party > 6 {
			return fmt.Errorf("party slot %d is outside the six-slot party", ref.Party)
		}
		data, err := poke.ToPartyData()
		if err != nil {
			return err
		}
		expanded, err := ParsePokemon(data, poke.NicknameRaw, poke.OTNameRaw)
		if err != nil {
			return err
		}
		for len(party) < ref.Party {
			party = append(party, nil)
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
			return fmt.Errorf("box slot %s is outside the 12 boxes of 20 slots", ref)
		}
		body, err := poke.ToBoxData()
		if err != nil {
			return err
		}
		base := boxListOffset(ref.Box)
		list := s.save.data[base : base+boxListSize]
		count := int(list[0])
		if ref.Slot > count+1 {
			return fmt.Errorf("box slot %s would create a gap (box has %d Pokémon)", ref, count)
		}
		bodyOff := 22 + (ref.Slot-1)*BoxRecordSize
		otOff := 22 + BoxCapacity*BoxRecordSize + (ref.Slot-1)*NameLength
		nickOff := otOff + BoxCapacity*NameLength
		copy(list[bodyOff:bodyOff+BoxRecordSize], body)
		copy(list[otOff:otOff+NameLength], poke.OTNameRaw)
		copy(list[nickOff:nickOff+NameLength], poke.NicknameRaw)
		list[1+ref.Slot-1] = body[0]
		if ref.Slot > count {
			list[0] = byte(ref.Slot)
		}
		list[1+BoxCapacity] = 0xFF
		s.save.recomputeChecksum()
		if index := int(s.save.data[currentIdx]); index == ref.Box-1 {
			copy(s.save.data[currentBox:currentBox+boxListSize], list)
			s.save.recomputeChecksum()
		}
		return nil
	default:
		return fmt.Errorf("unknown slot kind %q", ref.Kind)
	}
}

func (s *slotStorage) ListSlots(includeEmpty bool) ([]core.SlotEntry, error) {
	out := make([]core.SlotEntry, 0, 6+BoxCount*BoxCapacity)
	party, err := s.save.GetParty()
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
	for box := 1; box <= BoxCount; box++ {
		base := boxListOffset(box)
		list := s.save.data[base : base+boxListSize]
		count := int(list[0])
		for slot := 1; slot <= BoxCapacity; slot++ {
			ref := core.Box(box, slot)
			if slot > count || list[1+slot-1] == 0 || list[1+slot-1] == 0xFF {
				if includeEmpty {
					out = append(out, core.SlotEntry{Ref: ref, Empty: true})
				}
				continue
			}
			bodyOff := 22 + (slot-1)*BoxRecordSize
			otOff := 22 + BoxCapacity*BoxRecordSize + (slot-1)*NameLength
			nickOff := otOff + BoxCapacity*NameLength
			poke, err := ParsePokemon(list[bodyOff:bodyOff+BoxRecordSize], list[nickOff:nickOff+NameLength], list[otOff:otOff+NameLength])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", ref, err)
			}
			poke.BoxNum, poke.BoxSlot = box, slot
			out = append(out, core.SlotEntry{Ref: ref, Pokemon: poke})
		}
	}
	return out, nil
}

func (s *SaveFile) recomputeChecksum() {
	s.data[checksumOffset] = mainChecksum(s.data)
}

var _ = binary.BigEndian

package gen5

import (
	"encoding/binary"
	"fmt"
	"os"

	"pktrade/internal/core"
	"pktrade/internal/modern"
	"pktrade/internal/saveblocks"
)

const (
	saveSize      = 0x80000
	boxCount      = 24
	boxCapacity   = 30
	boxSlotSize   = 136
	partySlotSize = 220
)

type SaveFile struct {
	data    []byte
	family  string
	game    string
	table   saveblocks.Table
	party   saveblocks.Block
	boxes   [boxCount]saveblocks.Block
	trainer saveblocks.Block
}

func init() {
	core.RegisterSaveParser(load)
}

func load(data []byte) (core.SaveFile, error) {
	save, err := parse(data)
	if err != nil {
		return nil, err
	}
	return save, nil
}

func parse(data []byte) (*SaveFile, error) {
	if len(data) != saveSize {
		return nil, fmt.Errorf("not a gen 5 save")
	}
	candidates := []struct{ key, family, game string }{
		{"b2w2", "b2w2", "Black 2/White 2"},
		{"bw", "bw", "Black/White"},
	}
	var chosen *SaveFile
	for _, c := range candidates {
		t, ok := saveblocks.Get(c.key)
		if !ok || !validTable(data, t) {
			continue
		}
		s := &SaveFile{data: append([]byte(nil), data...), family: c.family, game: c.game, table: t}
		if err := s.indexBlocks(); err != nil {
			continue
		}
		// Prefer the table whose version byte matches its era when both
		// checksum sets happen to validate.
		version := s.data[s.trainer.Offset+0x1F]
		if c.key == "b2w2" && version >= 22 {
			chosen = s
			break
		}
		if c.key == "bw" && version < 22 {
			chosen = s
			break
		}
		if chosen == nil {
			chosen = s
		}
	}
	if chosen == nil {
		return nil, fmt.Errorf("not a gen 5 save (no valid Black/White or Black 2/White 2 block checksums)")
	}
	return chosen, nil
}

func validTable(data []byte, t saveblocks.Table) bool {
	if len(data) < t.Size {
		return false
	}
	for _, b := range t.Blocks {
		if b.Offset+b.Length > len(data) || b.Checksum+2 > len(data) || b.Mirror+2 > len(data) {
			return false
		}
		sum := crc16CCITT(data[b.Offset : b.Offset+b.Length])
		if binary.LittleEndian.Uint16(data[b.Checksum:]) != sum || binary.LittleEndian.Uint16(data[b.Mirror:]) != sum {
			return false
		}
	}
	return true
}

func (s *SaveFile) indexBlocks() error {
	for _, b := range s.table.Blocks {
		switch {
		case b.Name == "Party Pokémon":
			s.party = b
		case b.Name == "Trainer Data":
			s.trainer = b
		case len(b.Name) > 4 && b.Name[:4] == "Box ":
			var n int
			if _, err := fmt.Sscanf(b.Name, "Box %d", &n); err == nil && n >= 1 && n <= boxCount {
				s.boxes[n-1] = b
			}
		}
	}
	if s.party.Length == 0 || s.trainer.Length == 0 {
		return fmt.Errorf("missing party or trainer block")
	}
	for i := range s.boxes {
		if s.boxes[i].Length == 0 {
			return fmt.Errorf("missing box %d block", i+1)
		}
	}
	return nil
}

func crc16CCITT(data []byte) uint16 {
	top, bot := byte(0xFF), byte(0xFF)
	for _, b := range data {
		x := b ^ top
		x ^= x >> 4
		top = bot ^ (x >> 3) ^ (x << 4)
		bot = x ^ (x << 5)
	}
	return uint16(top)<<8 | uint16(bot)
}

func (s *SaveFile) Generation() int { return 5 }
func (s *SaveFile) Info() core.GameInfo {
	return core.GameInfo{Generation: 5, Family: s.family, Game: s.game}
}
func (s *SaveFile) CompatibilityKey() core.CompatibilityKey {
	return core.CompatibilityKey{Generation: 5, Family: s.family, Format: 5}
}
func (s *SaveFile) Capabilities() core.Capabilities {
	return core.Capabilities{Party: true, Boxes: true, Evolution: true, Dex: true, CrossGen: true}
}
func (s *SaveFile) Validate() error {
	if !validTable(s.data, s.table) {
		return fmt.Errorf("Gen 5 block checksum mismatch")
	}
	return nil
}
func (s *SaveFile) TrainerInfo() (core.TrainerInfo, error) {
	base := s.trainer.Offset
	return core.TrainerInfo{
		Name: modern.DecodeString(5, s.data[base+0x04:base+0x14]),
		TID:  binary.LittleEndian.Uint16(s.data[base+0x14:]),
		SID:  binary.LittleEndian.Uint16(s.data[base+0x16:]),
	}, nil
}

func (s *SaveFile) partyCount() int {
	n := int(s.data[s.party.Offset+4])
	if n < 0 {
		return 0
	}
	if n > 6 {
		return 6
	}
	return n
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	n := s.partyCount()
	out := make([]core.Pokemon, 0, n)
	for i := 0; i < n; i++ {
		off := s.party.Offset + 8 + i*partySlotSize
		if off+partySlotSize > len(s.data) {
			return nil, fmt.Errorf("party slot %d is outside the save", i+1)
		}
		p, err := modern.Parse(modern.Format5, s.data[off:off+partySlotSize])
		if err != nil {
			return nil, fmt.Errorf("party slot %d: %w", i+1, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	if len(party) > 6 {
		return fmt.Errorf("Gen 5 party cannot hold more than six Pokémon")
	}
	for i, member := range party {
		p, ok := member.(*modern.Pokemon)
		if !ok || p.Format.Generation != 5 {
			return fmt.Errorf("party slot %d is not a Gen 5 record", i+1)
		}
		raw, err := p.Encode(partySlotSize)
		if err != nil {
			return err
		}
		off := s.party.Offset + 8 + i*partySlotSize
		copy(s.data[off:off+partySlotSize], raw)
	}
	s.data[s.party.Offset+4] = byte(len(party))
	s.fixChecksums()
	return nil
}

func (s *SaveFile) PartyCapacity() int { return 6 }
func (s *SaveFile) BoxCount() int      { return boxCount }
func (s *SaveFile) BoxCapacity() int   { return boxCapacity }

type slotStorage struct{ save *SaveFile }

func (s *SaveFile) ReadSlot(ref core.SlotRef) (core.Pokemon, error) {
	return (&slotStorage{save: s}).ReadSlot(ref)
}
func (s *SaveFile) WriteSlot(ref core.SlotRef, incoming core.Pokemon) error {
	return (&slotStorage{save: s}).WriteSlot(ref, incoming)
}
func (s *SaveFile) ListSlots(includeEmpty bool) ([]core.SlotEntry, error) {
	return (&slotStorage{save: s}).ListSlots(includeEmpty)
}

func (st *slotStorage) ReadSlot(ref core.SlotRef) (core.Pokemon, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	s := st.save
	switch ref.Kind {
	case core.PartySlot:
		party, err := s.GetParty()
		if err != nil {
			return nil, err
		}
		if ref.Party < 1 || ref.Party > len(party) {
			return nil, fmt.Errorf("party slot %d is empty or does not exist", ref.Party)
		}
		return party[ref.Party-1], nil
	case core.BoxSlot:
		if ref.Box < 1 || ref.Box > boxCount || ref.Slot < 1 || ref.Slot > boxCapacity {
			return nil, fmt.Errorf("box slot %s is outside the 24 boxes of 30 slots", ref)
		}
		off := s.boxes[ref.Box-1].Offset + (ref.Slot-1)*boxSlotSize
		p, err := modern.Parse(modern.Format5, s.data[off:off+boxSlotSize])
		if err != nil {
			return nil, fmt.Errorf("box slot %s: %w", ref, err)
		}
		p.BoxNum, p.BoxSlot = ref.Box, ref.Slot
		return p, nil
	default:
		return nil, fmt.Errorf("unknown slot kind %q", ref.Kind)
	}
}

func (st *slotStorage) WriteSlot(ref core.SlotRef, incoming core.Pokemon) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	s := st.save
	p, ok := incoming.(*modern.Pokemon)
	if !ok || p.Format.Generation != 5 {
		return fmt.Errorf("cannot write a non-Gen 5 record; cross-generation writes are not supported")
	}
	switch ref.Kind {
	case core.PartySlot:
		party, err := s.GetParty()
		if err != nil {
			return err
		}
		if ref.Party < 1 || ref.Party > 6 {
			return fmt.Errorf("party slot %d is outside the six-slot party", ref.Party)
		}
		if ref.Party > len(party) {
			return fmt.Errorf("party slot %d is empty; party trades cannot create a gap", ref.Party)
		}
		party[ref.Party-1] = p
		return s.InjectParty(party)
	case core.BoxSlot:
		if ref.Box < 1 || ref.Box > boxCount || ref.Slot < 1 || ref.Slot > boxCapacity {
			return fmt.Errorf("box slot %s is outside the 24 boxes of 30 slots", ref)
		}
		raw, err := p.Encode(boxSlotSize)
		if err != nil {
			return err
		}
		off := s.boxes[ref.Box-1].Offset + (ref.Slot-1)*boxSlotSize
		copy(s.data[off:off+boxSlotSize], raw)
		s.fixChecksums()
		return nil
	default:
		return fmt.Errorf("unknown slot kind %q", ref.Kind)
	}
}

func (st *slotStorage) ListSlots(includeEmpty bool) ([]core.SlotEntry, error) {
	s := st.save
	out := make([]core.SlotEntry, 0, 6+boxCount*boxCapacity)
	party, err := s.GetParty()
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
	for box := 1; box <= boxCount; box++ {
		for slot := 1; slot <= boxCapacity; slot++ {
			ref := core.Box(box, slot)
			off := s.boxes[box-1].Offset + (slot-1)*boxSlotSize
			p, err := modern.Parse(modern.Format5, s.data[off:off+boxSlotSize])
			if err != nil {
				if includeEmpty {
					out = append(out, core.SlotEntry{Ref: ref, Empty: true})
				}
				continue
			}
			p.BoxNum, p.BoxSlot = box, slot
			out = append(out, core.SlotEntry{Ref: ref, Pokemon: p})
		}
	}
	return out, nil
}

func (s *SaveFile) fixChecksums() {
	for _, b := range s.table.Blocks {
		sum := crc16CCITT(s.data[b.Offset : b.Offset+b.Length])
		binary.LittleEndian.PutUint16(s.data[b.Checksum:], sum)
		binary.LittleEndian.PutUint16(s.data[b.Mirror:], sum)
	}
}

func (s *SaveFile) FixupReceived(incoming core.Pokemon) (core.Pokemon, []core.Evolution, error) {
	p, ok := incoming.(*modern.Pokemon)
	if !ok || p.Format.Generation != 5 {
		return nil, nil, fmt.Errorf("received record is not a Gen 5 Pokémon")
	}
	evolved, evo, err := modern.Evolve(p)
	if err != nil {
		return nil, nil, err
	}
	if evo == nil {
		return evolved, nil, nil
	}
	return evolved, []core.Evolution{*evo}, nil
}

func (s *SaveFile) WriteToFile(path string) error {
	return os.WriteFile(path, s.WriteToBytes(), 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	out := append([]byte(nil), s.data...)
	for _, b := range s.table.Blocks {
		sum := crc16CCITT(out[b.Offset : b.Offset+b.Length])
		binary.LittleEndian.PutUint16(out[b.Checksum:], sum)
		binary.LittleEndian.PutUint16(out[b.Mirror:], sum)
	}
	return out
}

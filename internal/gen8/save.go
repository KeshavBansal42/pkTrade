package gen8

import (
	"encoding/binary"
	"fmt"
	"os"

	"pktrade/internal/core"
	"pktrade/internal/modern"
)

// Sword/Shield stores its save as a list of key-addressed blocks that are
// XORed with a per-block keystream and signed with a SHA-256 hash over the
// whole payload. The hash is not a keyed MAC, so a modified save can be
// re-signed without any game-specific secret.
const (
	keyBox        = 0x0D66012C
	keyParty      = 0x2985FE5D
	keyMyStatus   = 0xF25C070E
	boxCount      = 32
	boxCapacity   = 30
	boxSlotSize   = 0x158
	partySlotSize = 0x158
)

type SaveFile struct {
	blocks []*scBlock
	byKey  map[uint32]*scBlock
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
	if len(data) < 0x100 {
		return nil, fmt.Errorf("not a gen 8 save")
	}
	blocks, err := decryptSwitch(data)
	if err != nil {
		return nil, fmt.Errorf("not a gen 8 save: %w", err)
	}
	s := &SaveFile{blocks: blocks, byKey: make(map[uint32]*scBlock, len(blocks))}
	for _, b := range blocks {
		s.byKey[b.Key] = b
	}
	for _, key := range []uint32{keyBox, keyParty, keyMyStatus} {
		if _, ok := s.byKey[key]; !ok {
			return nil, fmt.Errorf("not a Sword/Shield save (missing block %08X)", key)
		}
	}
	return s, nil
}

func (s *SaveFile) block(key uint32) (*scBlock, error) {
	b, ok := s.byKey[key]
	if !ok {
		return nil, fmt.Errorf("save is missing block %08X", key)
	}
	return b, nil
}

func (s *SaveFile) Generation() int { return 8 }
func (s *SaveFile) Info() core.GameInfo {
	return core.GameInfo{Generation: 8, Family: "swsh", Game: "Sword/Shield"}
}
func (s *SaveFile) CompatibilityKey() core.CompatibilityKey {
	return core.CompatibilityKey{Generation: 8, Family: "swsh", Format: 8}
}
func (s *SaveFile) Capabilities() core.Capabilities {
	return core.Capabilities{Party: true, Boxes: true, Evolution: true, Dex: false}
}
func (s *SaveFile) Validate() error {
	box, err := s.block(keyBox)
	if err != nil {
		return err
	}
	if len(box.Data) < boxCount*boxCapacity*boxSlotSize {
		return fmt.Errorf("Sword/Shield box block is smaller than %d slots", boxCount*boxCapacity)
	}
	party, err := s.block(keyParty)
	if err != nil {
		return err
	}
	if len(party.Data) < 6*partySlotSize+1 {
		return fmt.Errorf("Sword/Shield party block is too small")
	}
	return nil
}
func (s *SaveFile) TrainerInfo() (core.TrainerInfo, error) {
	b, err := s.block(keyMyStatus)
	if err != nil {
		return core.TrainerInfo{}, err
	}
	if len(b.Data) < 0xD0 {
		return core.TrainerInfo{}, fmt.Errorf("Sword/Shield status block is too small")
	}
	return core.TrainerInfo{
		Name:   modern.DecodeString(8, b.Data[0xB0:0xCA]),
		TID:    binary.LittleEndian.Uint16(b.Data[0xA0:]),
		SID:    binary.LittleEndian.Uint16(b.Data[0xA2:]),
		Gender: b.Data[0xA5],
	}, nil
}

func (s *SaveFile) partyCount() int {
	b, err := s.block(keyParty)
	if err != nil {
		return 0
	}
	index := 6 * partySlotSize
	if index >= len(b.Data) {
		return 0
	}
	n := int(b.Data[index])
	if n > 6 {
		return 6
	}
	if n < 0 {
		return 0
	}
	return n
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	b, err := s.block(keyParty)
	if err != nil {
		return nil, err
	}
	n := s.partyCount()
	out := make([]core.Pokemon, 0, n)
	for i := 0; i < n; i++ {
		off := i * partySlotSize
		if off+partySlotSize > len(b.Data) {
			return nil, fmt.Errorf("party slot %d is outside the save", i+1)
		}
		p, err := modern.Parse(modern.Format8, b.Data[off:off+partySlotSize])
		if err != nil {
			return nil, fmt.Errorf("party slot %d: %w", i+1, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	if len(party) > 6 {
		return fmt.Errorf("Gen 8 party cannot hold more than six Pokémon")
	}
	b, err := s.block(keyParty)
	if err != nil {
		return err
	}
	for i, member := range party {
		p, ok := member.(*modern.Pokemon)
		if !ok || p.Format.Generation != 8 {
			return fmt.Errorf("party slot %d is not a Gen 8 record", i+1)
		}
		raw, err := p.Encode(partySlotSize)
		if err != nil {
			return err
		}
		off := i * partySlotSize
		if off+partySlotSize > len(b.Data) {
			return fmt.Errorf("party slot %d is outside the save", i+1)
		}
		copy(b.Data[off:off+partySlotSize], raw)
	}
	b.Data[6*partySlotSize] = byte(len(party))
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
			return nil, fmt.Errorf("box slot %s is outside the 32 boxes of 30 slots", ref)
		}
		b, err := s.block(keyBox)
		if err != nil {
			return nil, err
		}
		off := ((ref.Box-1)*boxCapacity + (ref.Slot - 1)) * boxSlotSize
		if off+boxSlotSize > len(b.Data) {
			return nil, fmt.Errorf("box slot %s is outside the save", ref)
		}
		p, err := modern.Parse(modern.Format8, b.Data[off:off+boxSlotSize])
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
	if !ok || p.Format.Generation != 8 {
		return fmt.Errorf("cannot write a non-Gen 8 record; cross-generation writes are not supported")
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
			return fmt.Errorf("box slot %s is outside the 32 boxes of 30 slots", ref)
		}
		b, err := s.block(keyBox)
		if err != nil {
			return err
		}
		raw, err := p.Encode(boxSlotSize)
		if err != nil {
			return err
		}
		off := ((ref.Box-1)*boxCapacity + (ref.Slot - 1)) * boxSlotSize
		if off+boxSlotSize > len(b.Data) {
			return fmt.Errorf("box slot %s is outside the save", ref)
		}
		copy(b.Data[off:off+boxSlotSize], raw)
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
	b, err := s.block(keyBox)
	if err != nil {
		return nil, err
	}
	for box := 1; box <= boxCount; box++ {
		for slot := 1; slot <= boxCapacity; slot++ {
			ref := core.Box(box, slot)
			off := ((box-1)*boxCapacity + (slot - 1)) * boxSlotSize
			if off+boxSlotSize > len(b.Data) {
				if includeEmpty {
					out = append(out, core.SlotEntry{Ref: ref, Empty: true})
				}
				continue
			}
			p, err := modern.Parse(modern.Format8, b.Data[off:off+boxSlotSize])
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

func (s *SaveFile) FixupReceived(incoming core.Pokemon) (core.Pokemon, []core.Evolution, error) {
	p, ok := incoming.(*modern.Pokemon)
	if !ok || p.Format.Generation != 8 {
		return nil, nil, fmt.Errorf("received record is not a Gen 8 Pokémon")
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

func (s *SaveFile) MarkDex(core.Pokemon) error { return nil }

func (s *SaveFile) WriteToFile(path string) error {
	return os.WriteFile(path, s.WriteToBytes(), 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	return encryptSwitch(s.blocks)
}

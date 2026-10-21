package gen4

import (
	"encoding/binary"
	"fmt"
	"os"

	"pktrade/internal/core"
	"pktrade/internal/modern"
)

const (
	saveSize       = 0x80000
	partitionSize  = 0x40000
	boxCount       = 18
	boxCapacity    = 30
	boxSlotSize    = 136
	partySlotSize  = 236
	footerSize     = 0x14
	partyCountSize = 6
)

type gameFormat struct {
	family       string
	game         string
	generalSize  int
	storageSize  int
	storageStart int
	footerSize   int
	trainer      int
	party        int
	boxStart     int
	boxStride    int
}

var formats = []gameFormat{
	{"dp", "Diamond/Pearl", 0xC100, 0x121E0, 0xC100, 0x14, 0x64, 0x98, 4, 30 * 136},
	{"pt", "Platinum", 0xCF2C, 0x121E4, 0xCF2C, 0x14, 0x68, 0xA0, 4, 30 * 136},
	{"hgss", "HeartGold/SoulSilver", 0xF628, 0x12310, 0xF628 + 0xD8, 0x10, 0x64, 0x98, 0, 0x1000},
}

type SaveFile struct {
	data        []byte
	format      gameFormat
	generalBase int
	storageBase int
}

// recordFormat returns the record layout of the game. HeartGold and
// SoulSilver place a few original-trainer fields at different offsets than
// Diamond, Pearl and Platinum.
func (s *SaveFile) recordFormat() modern.Format {
	if s.format.family == "hgss" {
		return modern.Format4HGSS
	}
	return modern.Format4
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
		return nil, fmt.Errorf("not a gen 4 save")
	}
	for _, f := range formats {
		s := &SaveFile{data: append([]byte(nil), data...), format: f}
		s.generalBase = s.activeBase(0, f.generalSize)
		s.storageBase = s.activeBase(f.storageStart, f.storageSize)
		if s.validChecksums() {
			return s, nil
		}
	}
	return nil, fmt.Errorf("not a gen 4 save (no valid Diamond/Pearl, Platinum, or HGSS checksums)")
}

func (s *SaveFile) activeBase(start, length int) int {
	footer := start + length - 0x14
	newest := compareFooters(s.data, footer, footer+partitionSize)
	if newest == 0 {
		return start
	}
	return start + partitionSize
}

func compareFooters(data []byte, a, b int) int {
	major := compareCounter(binary.LittleEndian.Uint32(data[a:]), binary.LittleEndian.Uint32(data[b:]))
	if major != 2 {
		return major
	}
	minor := compareCounter(binary.LittleEndian.Uint32(data[a+4:]), binary.LittleEndian.Uint32(data[b+4:]))
	if minor == 1 {
		return 1
	}
	return 0
}

func compareCounter(a, b uint32) int {
	if a == 0xFFFFFFFF && b != 0xFFFFFFFE {
		return 1
	}
	if b == 0xFFFFFFFF && a != 0xFFFFFFFE {
		return 0
	}
	if a > b {
		return 0
	}
	if a < b {
		return 1
	}
	return 2
}

func (s *SaveFile) validChecksums() bool {
	return s.blockChecksum(s.generalBase, s.format.generalSize) == binary.LittleEndian.Uint16(s.data[s.generalBase+s.format.generalSize-2:]) &&
		s.blockChecksum(s.storageBase, s.format.storageSize) == binary.LittleEndian.Uint16(s.data[s.storageBase+s.format.storageSize-2:])
}

func (s *SaveFile) blockChecksum(base, length int) uint16 {
	return crc16CCITT(s.data[base : base+length-s.format.footerSize])
}

func (s *SaveFile) fixChecksums(out []byte) {
	binary.LittleEndian.PutUint16(out[s.generalBase+s.format.generalSize-2:], crc16CCITT(out[s.generalBase:s.generalBase+s.format.generalSize-s.format.footerSize]))
	binary.LittleEndian.PutUint16(out[s.storageBase+s.format.storageSize-2:], crc16CCITT(out[s.storageBase:s.storageBase+s.format.storageSize-s.format.footerSize]))
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

func (s *SaveFile) Generation() int { return 4 }
func (s *SaveFile) Info() core.GameInfo {
	return core.GameInfo{Generation: 4, Family: s.format.family, Game: s.format.game, ActiveSlot: fmt.Sprintf("0x%X", s.generalBase)}
}
func (s *SaveFile) CompatibilityKey() core.CompatibilityKey {
	return core.CompatibilityKey{Generation: 4, Family: s.format.family, Format: 4}
}
func (s *SaveFile) Capabilities() core.Capabilities {
	return core.Capabilities{Party: true, Boxes: true, Evolution: true, Dex: true}
}
func (s *SaveFile) Validate() error {
	if !s.validChecksums() {
		return fmt.Errorf("Gen 4 save checksum mismatch")
	}
	return nil
}

func (s *SaveFile) TrainerInfo() (core.TrainerInfo, error) {
	base := s.generalBase + s.format.trainer
	name := modern.DecodeString(4, s.data[base:base+16])
	return core.TrainerInfo{
		Name: name,
		TID:  binary.LittleEndian.Uint16(s.data[base+0x10:]),
		SID:  binary.LittleEndian.Uint16(s.data[base+0x12:]),
	}, nil
}

func (s *SaveFile) partyCount() int {
	n := int(s.data[s.generalBase+s.format.party-4])
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
		off := s.generalBase + s.format.party + i*partySlotSize
		p, err := modern.Parse(s.recordFormat(), s.data[off:off+partySlotSize])
		if err != nil {
			return nil, fmt.Errorf("party slot %d: %w", i+1, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	if len(party) > 6 {
		return fmt.Errorf("Gen 4 party cannot hold more than six Pokémon")
	}
	for i, member := range party {
		p, ok := member.(*modern.Pokemon)
		if !ok || p.Format.Generation != 4 {
			return fmt.Errorf("party slot %d is not a Gen 4 record", i+1)
		}
		raw, err := p.Encode(partySlotSize)
		if err != nil {
			return err
		}
		off := s.generalBase + s.format.party + i*partySlotSize
		copy(s.data[off:off+partySlotSize], raw)
	}
	s.data[s.generalBase+s.format.party-4] = byte(len(party))
	s.fixChecksums(s.data)
	return nil
}

func (s *SaveFile) PartyCapacity() int { return 6 }
func (s *SaveFile) BoxCount() int      { return boxCount }
func (s *SaveFile) BoxCapacity() int   { return boxCapacity }

func (s *SaveFile) boxOffset(box, slot int) int {
	return s.storageBase + s.format.boxStart + box*s.format.boxStride + slot*boxSlotSize
}

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
			return nil, fmt.Errorf("box slot %s is outside the 18 boxes of 30 slots", ref)
		}
		off := s.boxOffset(ref.Box-1, ref.Slot-1)
		p, err := modern.Parse(s.recordFormat(), s.data[off:off+boxSlotSize])
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
	if !ok || p.Format.Generation != 4 {
		return fmt.Errorf("cannot write a non-Gen 4 record; cross-generation writes are not supported")
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
		for len(party) < ref.Party {
			return fmt.Errorf("party slot %d is empty; party trades cannot create a gap", ref.Party)
		}
		party[ref.Party-1] = p
		return s.InjectParty(party)
	case core.BoxSlot:
		if ref.Box < 1 || ref.Box > boxCount || ref.Slot < 1 || ref.Slot > boxCapacity {
			return fmt.Errorf("box slot %s is outside the 18 boxes of 30 slots", ref)
		}
		raw, err := p.Encode(boxSlotSize)
		if err != nil {
			return err
		}
		off := s.boxOffset(ref.Box-1, ref.Slot-1)
		copy(s.data[off:off+boxSlotSize], raw)
		s.fixChecksums(s.data)
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
			off := s.boxOffset(box-1, slot-1)
			p, err := modern.Parse(s.recordFormat(), s.data[off:off+boxSlotSize])
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
	if !ok || p.Format.Generation != 4 {
		return nil, nil, fmt.Errorf("received record is not a Gen 4 Pokémon")
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
	s.fixChecksums(out)
	return out
}

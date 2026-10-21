package gen7

import (
	"encoding/binary"
	"fmt"
	"os"

	"pktrade/internal/core"
	"pktrade/internal/modern"
	"pktrade/internal/saveblocks"
)

const (
	boxCount      = 32
	boxCapacity   = 30
	boxSlotSize   = 0xE8
	partySlotSize = 0x104
)

type format struct {
	key, family, game        string
	status, party, box       int
	fixedParty, fixedBox     int
	statusOT, statusLanguage int
}

var formats = []format{
	{"sm", "sm", "Sun/Moon", 3, 4, 14, -1, -1, 0x38, 0x35},
	{"usum", "usum", "Ultra Sun/Ultra Moon", 3, 4, 14, -1, -1, 0x38, 0x35},
}

type SaveFile struct {
	data       []byte
	format     format
	table      saveblocks.Table
	partyBase  int
	boxBase    int
	statusBase int
	signature  int
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
	for _, f := range formats {
		t, ok := saveblocks.Get(f.key)
		if !ok || len(data) != t.Size {
			continue
		}
		if !validTable(data, t, f.key) {
			continue
		}
		s := &SaveFile{data: append([]byte(nil), data...), format: f, table: t, signature: -1}
		if f.key == "sm" || f.key == "usum" {
			s.signature = 36
		}
		if f.fixedParty >= 0 {
			s.partyBase = f.fixedParty
		} else if f.party >= 0 && f.party < len(t.Blocks) {
			s.partyBase = t.Blocks[f.party].Offset
		}
		if f.fixedBox >= 0 {
			s.boxBase = f.fixedBox
		} else if f.box >= 0 && f.box < len(t.Blocks) {
			s.boxBase = t.Blocks[f.box].Offset
		}
		if f.status < 0 || f.status >= len(t.Blocks) || s.partyBase == 0 || s.boxBase == 0 {
			continue
		}
		s.statusBase = t.Blocks[f.status].Offset
		return s, nil
	}
	return nil, fmt.Errorf("not a gen 7 save (no valid Sun/Moon or Ultra Sun/Ultra Moon block checksums)")
}

func validTable(data []byte, t saveblocks.Table, key string) bool {
	if len(data) < t.Size {
		return false
	}
	for i, b := range t.Blocks {
		if i == 36 && (key == "sm" || key == "usum") {
			continue // MemeCrypto signature block has a deliberately stale checksum.
		}
		if b.Offset+b.Length > len(data) {
			return false
		}
		off := t.MetadataOffset + 0x14 + b.ID*8 + 6
		if off+2 > len(data) {
			return false
		}
		if binary.LittleEndian.Uint16(data[off:]) != blockChecksum(t.Checksum, data[b.Offset:b.Offset+b.Length]) {
			return false
		}
	}
	return true
}

func blockChecksum(algorithm string, data []byte) uint16 {
	switch algorithm {
	case "crc16_ccitt":
		return crc16CCITT(data)
	case "crc16_invert":
		return crc16Invert(data)
	case "crc16_noinvert":
		return crc16NoInvert(data)
	}
	return 0
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

var crcTable = func() [256]uint16 {
	var t [256]uint16
	for i := 0; i < 256; i++ {
		v := uint16(i)
		for j := 0; j < 8; j++ {
			if v&1 != 0 {
				v = (v >> 1) ^ 0xA001
			} else {
				v >>= 1
			}
		}
		t[i] = v
	}
	return t
}()

func crc16Reflect(data []byte, initial uint16) uint16 {
	c := initial
	for _, b := range data {
		c = crcTable[byte(c)^b] ^ (c >> 8)
	}
	return c
}

func crc16Invert(data []byte) uint16   { return ^crc16Reflect(data, 0xFFFF) }
func crc16NoInvert(data []byte) uint16 { return crc16Reflect(data, 0xFFFF) }

func (s *SaveFile) Generation() int { return 7 }
func (s *SaveFile) Info() core.GameInfo {
	return core.GameInfo{Generation: 7, Family: s.format.family, Game: s.format.game}
}
func (s *SaveFile) CompatibilityKey() core.CompatibilityKey {
	return core.CompatibilityKey{Generation: 7, Family: s.format.family, Format: 7}
}
func (s *SaveFile) Capabilities() core.Capabilities {
	return core.Capabilities{Party: true, Boxes: true, Evolution: true, Dex: true}
}
func (s *SaveFile) Validate() error {
	if !validTable(s.data, s.table, s.format.key) {
		return fmt.Errorf("Gen 6 block checksum mismatch")
	}
	return nil
}
func (s *SaveFile) TrainerInfo() (core.TrainerInfo, error) {
	base := s.statusBase
	return core.TrainerInfo{
		Name: modern.DecodeString(7, s.data[base+s.format.statusOT:base+s.format.statusOT+0x1A]),
		TID:  binary.LittleEndian.Uint16(s.data[base:]),
		SID:  binary.LittleEndian.Uint16(s.data[base+2:]),
	}, nil
}

func (s *SaveFile) partyCount() int {
	off := s.partyBase + 6*partySlotSize
	if off >= len(s.data) {
		return 0
	}
	n := int(s.data[off])
	if n > 6 {
		return 6
	}
	if n < 0 {
		return 0
	}
	return n
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	n := s.partyCount()
	out := make([]core.Pokemon, 0, n)
	for i := 0; i < n; i++ {
		off := s.partyBase + i*partySlotSize
		p, err := modern.Parse(modern.Format7, s.data[off:off+partySlotSize])
		if err != nil {
			return nil, fmt.Errorf("party slot %d: %w", i+1, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	if len(party) > 6 {
		return fmt.Errorf("Gen 7 party cannot hold more than six Pokémon")
	}
	for i, member := range party {
		p, ok := member.(*modern.Pokemon)
		if !ok || p.Format.Generation != 7 {
			return fmt.Errorf("party slot %d is not a Gen 7 record", i+1)
		}
		raw, err := p.Encode(partySlotSize)
		if err != nil {
			return err
		}
		off := s.partyBase + i*partySlotSize
		copy(s.data[off:off+partySlotSize], raw)
	}
	s.data[s.partyBase+6*partySlotSize] = byte(len(party))
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
			return nil, fmt.Errorf("box slot %s is outside the 32 boxes of 30 slots", ref)
		}
		off := s.boxBase + ((ref.Box-1)*boxCapacity+(ref.Slot-1))*boxSlotSize
		p, err := modern.Parse(modern.Format7, s.data[off:off+boxSlotSize])
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
	if !ok || p.Format.Generation != 7 {
		return fmt.Errorf("cannot write a non-Gen 6 record; cross-generation writes are not supported")
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
		raw, err := p.Encode(boxSlotSize)
		if err != nil {
			return err
		}
		off := s.boxBase + ((ref.Box-1)*boxCapacity+(ref.Slot-1))*boxSlotSize
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
			off := s.boxBase + ((box-1)*boxCapacity+(slot-1))*boxSlotSize
			p, err := modern.Parse(modern.Format7, s.data[off:off+boxSlotSize])
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
	for i, b := range s.table.Blocks {
		if i == s.signature {
			continue
		}
		sum := blockChecksum(s.table.Checksum, s.data[b.Offset:b.Offset+b.Length])
		off := s.table.MetadataOffset + 0x14 + b.ID*8 + 6
		binary.LittleEndian.PutUint16(s.data[off:], sum)
	}
}

func (s *SaveFile) FixupReceived(incoming core.Pokemon) (core.Pokemon, []core.Evolution, error) {
	p, ok := incoming.(*modern.Pokemon)
	if !ok || p.Format.Generation != 7 {
		return nil, nil, fmt.Errorf("received record is not a Gen 7 Pokémon")
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
	for i, b := range s.table.Blocks {
		if i == s.signature {
			continue
		}
		sum := blockChecksum(s.table.Checksum, out[b.Offset:b.Offset+b.Length])
		off := s.table.MetadataOffset + 0x14 + b.ID*8 + 6
		binary.LittleEndian.PutUint16(out[off:], sum)
	}
	return out
}

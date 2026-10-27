package gen1

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

// PartyRecordSize is the size of a party Pokémon's core data structure.
const PartyRecordSize = 44

// BoxRecordSize is the stored (boxed) record size. The last eleven bytes of a
// party record are party-only level and calculated stats.
const BoxRecordSize = 33

// NameLength is the fixed width of the OT name and nickname fields.
const NameLength = 11

// Pokemon represents a Generation I Pokémon in either party or boxed form.
// RawData keeps the exact on-disk record and the name fields keep their exact
// raw bytes so a trade can move them without re-encoding.
type Pokemon struct {
	RawData []byte

	NicknameRaw []byte
	OTNameRaw   []byte

	SpeciesID uint8
	CurrentHP uint16
	LevelVal  uint8
	Status    uint8
	Type1     uint8
	Type2     uint8
	CatchRate uint8
	MoveList  [4]uint8
	OTID      uint16
	Exp       uint32
	DVs       uint16
	PPMoves   [4]uint8

	NicknameStr string
	OTNameStr   string

	Stored  bool
	IsEgg   bool
	BoxNum  int
	BoxSlot int
}

// ParsePokemon parses a 33-byte boxed or 44-byte party record plus its
// nickname and OT name arrays.
func ParsePokemon(data, nickname, otname []byte) (*Pokemon, error) {
	if len(data) != BoxRecordSize && len(data) != PartyRecordSize {
		return nil, fmt.Errorf("gen 1 record must be 33 or 44 bytes, got %d", len(data))
	}
	if len(nickname) != NameLength || len(otname) != NameLength {
		return nil, fmt.Errorf("gen 1 name fields must be %d bytes", NameLength)
	}

	p := &Pokemon{
		RawData:     append([]byte(nil), data...),
		NicknameRaw: append([]byte(nil), nickname...),
		OTNameRaw:   append([]byte(nil), otname...),
		Stored:      len(data) == BoxRecordSize,
	}
	p.SpeciesID = data[0x00]
	p.CurrentHP = binary.BigEndian.Uint16(data[0x01:0x03])
	if p.Stored {
		p.LevelVal = data[0x03]
	} else {
		p.LevelVal = data[0x21]
	}
	p.Status = data[0x04]
	p.Type1 = data[0x05]
	p.Type2 = data[0x06]
	p.CatchRate = data[0x07]
	p.MoveList = [4]uint8{data[0x08], data[0x09], data[0x0A], data[0x0B]}
	p.OTID = binary.BigEndian.Uint16(data[0x0C:0x0E])
	p.Exp = binary.BigEndian.Uint32([]byte{0, data[0x0E], data[0x0F], data[0x10]})
	p.DVs = binary.BigEndian.Uint16(data[0x1B:0x1D])
	p.PPMoves = [4]uint8{data[0x1D], data[0x1E], data[0x1F], data[0x20]}

	p.NicknameStr = decodeGen1String(p.NicknameRaw)
	p.OTNameStr = decodeGen1String(p.OTNameRaw)
	return p, nil
}

func (p *Pokemon) GetSpecies() uint16  { return uint16(p.SpeciesID) }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte      { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 {
	return [4]uint16{
		uint16(p.MoveList[0]), uint16(p.MoveList[1]),
		uint16(p.MoveList[2]), uint16(p.MoveList[3]),
	}
}
func (p *Pokemon) GetHeldItem() uint16    { return 0 } // Gen 1 has no held items
func (p *Pokemon) GetTrainerName() string { return p.OTNameStr }
func (p *Pokemon) GetTrainerID() uint32   { return uint32(p.OTID) }
func (p *Pokemon) GetGeneration() int     { return 1 }
func (p *Pokemon) ToBytes() []byte        { return append([]byte(nil), p.RawData...) }
func (p *Pokemon) IsChecksumValid() bool  { return true } // no per-record checksum

// NationalID returns the National Dex number for this record.
func (p *Pokemon) NationalID() uint16 { return pokemon.National1(uint16(p.SpeciesID)) }

// TradeBlockReason rejects invalid glitch records. Generation I has no eggs.
func (p *Pokemon) TradeBlockReason() string {
	if p.SpeciesID == 0 {
		return "the record has no species"
	}
	if p.NationalID() == 0 {
		return "the record's species has no National Dex mapping"
	}
	if p.LevelVal == 0 || p.LevelVal > 100 {
		return "the record's level is invalid"
	}
	return ""
}

func (p *Pokemon) clone() *Pokemon {
	out := *p
	out.RawData = append([]byte(nil), p.RawData...)
	out.NicknameRaw = append([]byte(nil), p.NicknameRaw...)
	out.OTNameRaw = append([]byte(nil), p.OTNameRaw...)
	return &out
}

// ToPartyData returns a 44-byte party record, calculating the party-only level
// and stats from the stored base stats, DVs, and stat experience.
func (p *Pokemon) ToPartyData() ([]byte, error) {
	if len(p.RawData) == PartyRecordSize {
		return append([]byte(nil), p.RawData...), nil
	}
	if len(p.RawData) != BoxRecordSize {
		return nil, fmt.Errorf("gen 1 record is truncated")
	}
	out := make([]byte, PartyRecordSize)
	copy(out, p.RawData)
	out[0x21] = out[0x03]
	stats, err := p.partyStats()
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint16(out[0x22:], stats[0])
	binary.BigEndian.PutUint16(out[0x24:], stats[1])
	binary.BigEndian.PutUint16(out[0x26:], stats[2])
	binary.BigEndian.PutUint16(out[0x28:], stats[3])
	binary.BigEndian.PutUint16(out[0x2A:], stats[4])
	return out, nil
}

// ToBoxData returns the 33-byte stored form. The party-only bytes are dropped
// and the level is copied into the boxed level field.
func (p *Pokemon) ToBoxData() ([]byte, error) {
	if len(p.RawData) == BoxRecordSize {
		return append([]byte(nil), p.RawData...), nil
	}
	if len(p.RawData) != PartyRecordSize {
		return nil, fmt.Errorf("gen 1 record is truncated")
	}
	out := append([]byte(nil), p.RawData[:BoxRecordSize]...)
	out[0x03] = p.RawData[0x21]
	return out, nil
}

func (p *Pokemon) partyStats() ([6]uint16, error) {
	national := p.NationalID()
	if national == 0 || int(national) >= len(pokemon.BaseStats1) {
		return [6]uint16{}, fmt.Errorf("unknown species %d", p.SpeciesID)
	}
	base := pokemon.BaseStats1[national]
	level := int(p.LevelVal)
	if level < 1 {
		level = 1
	}
	if level > 100 {
		level = 100
	}
	dv := p.DVs
	atk := int(dv & 0xF)
	def := int((dv >> 4) & 0xF)
	spe := int((dv >> 8) & 0xF)
	spc := int((dv >> 12) & 0xF)
	hpDV := ((atk & 1) << 3) | ((def & 1) << 2) | ((spe & 1) << 1) | (spc & 1)

	statExp := func(offset int) int {
		return int(binary.BigEndian.Uint16(p.RawData[offset:]))
	}
	calc := func(base, dv, exp int, hp bool) int {
		value := ((base+dv)*2 + int(math.Sqrt(float64(exp)))/4) * level / 100
		if hp {
			return value + level + 10
		}
		return value + 5
	}
	return [6]uint16{
		uint16(calc(base.HP, hpDV, statExp(0x11), true)),
		uint16(calc(base.Atk, atk, statExp(0x13), false)),
		uint16(calc(base.Def, def, statExp(0x15), false)),
		uint16(calc(base.Spe, spe, statExp(0x17), false)),
		uint16(calc(base.SpA, spc, statExp(0x19), false)),
		uint16(calc(base.SpD, spc, statExp(0x19), false)),
	}, nil
}

var _ core.Pokemon = (*Pokemon)(nil)

// gen1Charset maps the Western Generation I character codes that appear in
// names to their text equivalents. Codes outside this set are dropped.
var gen1Charset = func() map[byte]string {
	m := map[byte]string{
		0x7F: " ",
		0x9A: "(", 0x9B: ")", 0x9C: ":", 0x9D: ";", 0x9E: "[", 0x9F: "]",
		0xC0: "Ä", 0xC1: "Ö", 0xC2: "Ü", 0xC3: "ä", 0xC4: "ö", 0xC5: "ü",
		0xE0: "'", 0xE1: "PK", 0xE2: "MN", 0xE3: "-", 0xE6: "?", 0xE7: "!", 0xE8: ".",
		0xF0: "\u20AC", 0xF1: "×", 0xF2: ".", 0xF3: "/", 0xF4: ",", 0xF5: "♀",
	}
	for i := byte(0); i < 26; i++ {
		m[0x80+i] = string(rune('A' + i))
		m[0xA0+i] = string(rune('a' + i))
	}
	for i := byte(0); i < 10; i++ {
		m[0xF6+i] = string(rune('0' + i))
	}
	return m
}()

// DecodeString converts a Generation I name field to text.
func DecodeString(data []byte) string { return decodeGen1String(data) }

// EncodeString converts text to a Generation I name field of the given width.
func EncodeString(value string, length int) []byte { return encodeGen1String(value, length) }

func decodeGen1String(data []byte) string {
	var b strings.Builder
	for _, c := range data {
		if c == 0x50 || c == 0x00 {
			break
		}
		if s, ok := gen1Charset[c]; ok {
			b.WriteString(s)
		}
	}
	return b.String()
}

func encodeGen1String(value string, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = 0x50
	}
	for i := 0; i < len(value) && i < length; i++ {
		c := value[i]
		switch {
		case c >= 'A' && c <= 'Z':
			out[i] = 0x80 + (c - 'A')
		case c >= 'a' && c <= 'z':
			out[i] = 0xA0 + (c - 'a')
		case c >= '0' && c <= '9':
			out[i] = 0xF6 + (c - '0')
		case c == ' ':
			out[i] = 0x7F
		case c == '-':
			out[i] = 0xE3
		case c == '.':
			out[i] = 0xE8
		default:
			out[i] = 0x7F
		}
	}
	return out
}

package gen2

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

// PartyRecordSize is the size of a party Pokémon's data structure in Gen 2.
const PartyRecordSize = 48

// BoxRecordSize is the stored (boxed) record size.
const BoxRecordSize = 32

// NameLength is the fixed width of the OT name and nickname fields.
const NameLength = 11

// EggMarker is the species-list marker used for eggs in Gen 2 boxes and party.
const EggMarker = 0xFD

// Pokemon represents a Generation II Pokémon in party or boxed form.
type Pokemon struct {
	RawData []byte

	NicknameRaw []byte
	OTNameRaw   []byte

	SpeciesID  uint8
	HeldItem   uint8
	Moves      [4]uint8
	OTID       uint16
	Exp        uint32
	DVs        uint16
	PPMoves    [4]uint8
	Friendship uint8
	Pokerus    uint8
	LevelVal   uint8

	NicknameStr string
	OTNameStr   string

	Stored  bool
	IsEgg   bool
	BoxNum  int
	BoxSlot int
}

func ParsePokemon(data, nickname, otname []byte) (*Pokemon, error) {
	return ParsePokemonMarker(data, nickname, otname, 0)
}

// ParsePokemonMarker parses a 32-byte boxed or 48-byte party record. The
// marker is the species-list entry, which is 0xFD for eggs.
func ParsePokemonMarker(data, nickname, otname []byte, marker byte) (*Pokemon, error) {
	if len(data) != BoxRecordSize && len(data) != PartyRecordSize {
		return nil, fmt.Errorf("gen 2 record must be 32 or 48 bytes, got %d", len(data))
	}
	if len(nickname) != NameLength || len(otname) != NameLength {
		return nil, fmt.Errorf("gen 2 name fields must be %d bytes", NameLength)
	}

	p := &Pokemon{
		RawData:     append([]byte(nil), data...),
		NicknameRaw: append([]byte(nil), nickname...),
		OTNameRaw:   append([]byte(nil), otname...),
		Stored:      len(data) == BoxRecordSize,
		IsEgg:       marker == EggMarker,
	}
	p.SpeciesID = data[0x00]
	p.HeldItem = data[0x01]
	p.Moves = [4]uint8{data[0x02], data[0x03], data[0x04], data[0x05]}
	p.OTID = binary.BigEndian.Uint16(data[0x06:0x08])
	p.Exp = binary.BigEndian.Uint32([]byte{0, data[0x08], data[0x09], data[0x0A]})
	p.DVs = binary.BigEndian.Uint16(data[0x15:0x17])
	p.PPMoves = [4]uint8{data[0x17], data[0x18], data[0x19], data[0x1A]}
	p.Friendship = data[0x1B]
	p.Pokerus = data[0x1C]
	p.LevelVal = data[0x1F]

	p.NicknameStr = decodeGen2String(p.NicknameRaw)
	p.OTNameStr = decodeGen2String(p.OTNameRaw)
	return p, nil
}

func (p *Pokemon) GetSpecies() uint16  { return uint16(p.SpeciesID) }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte      { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 {
	return [4]uint16{
		uint16(p.Moves[0]), uint16(p.Moves[1]),
		uint16(p.Moves[2]), uint16(p.Moves[3]),
	}
}
func (p *Pokemon) GetHeldItem() uint16    { return uint16(p.HeldItem) }
func (p *Pokemon) GetTrainerName() string { return p.OTNameStr }
func (p *Pokemon) GetTrainerID() uint32   { return uint32(p.OTID) }
func (p *Pokemon) GetGeneration() int     { return 2 }
func (p *Pokemon) ToBytes() []byte        { return append([]byte(nil), p.RawData...) }
func (p *Pokemon) IsChecksumValid() bool  { return true }

func (p *Pokemon) NationalID() uint16 {
	if p.SpeciesID == 0 || p.SpeciesID > 251 {
		return 0
	}
	return uint16(p.SpeciesID)
}

func (p *Pokemon) TradeBlockReason() string {
	if p.SpeciesID == 0 {
		return "the record has no species"
	}
	if p.IsEgg {
		return "eggs cannot be traded"
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

func (p *Pokemon) ToPartyData() ([]byte, error) {
	if len(p.RawData) == PartyRecordSize {
		return append([]byte(nil), p.RawData...), nil
	}
	if len(p.RawData) != BoxRecordSize {
		return nil, fmt.Errorf("gen 2 record is truncated")
	}
	out := make([]byte, PartyRecordSize)
	copy(out, p.RawData)
	out[0x20] = 0 // status
	out[0x21] = 0 // mail
	stats, err := p.partyStats()
	if err != nil {
		return nil, err
	}
	put := func(offset int, value uint16) { binary.BigEndian.PutUint16(out[offset:], value) }
	put(0x22, stats[0])
	put(0x24, stats[0])
	put(0x26, stats[1])
	put(0x28, stats[2])
	put(0x2A, stats[3])
	put(0x2C, stats[4])
	put(0x2E, stats[5])
	return out, nil
}

func (p *Pokemon) ToBoxData() ([]byte, error) {
	if len(p.RawData) == BoxRecordSize {
		return append([]byte(nil), p.RawData...), nil
	}
	if len(p.RawData) != PartyRecordSize {
		return nil, fmt.Errorf("gen 2 record is truncated")
	}
	return append([]byte(nil), p.RawData[:BoxRecordSize]...), nil
}

func (p *Pokemon) partyStats() ([6]uint16, error) {
	national := p.NationalID()
	if national == 0 || int(national) >= len(pokemon.BaseStats2) {
		return [6]uint16{}, fmt.Errorf("unknown species %d", p.SpeciesID)
	}
	base := pokemon.BaseStats2[national]
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
	statExp := func(offset int) int { return int(binary.BigEndian.Uint16(p.RawData[offset:])) }
	calc := func(base, dv, exp int, hp bool) int {
		value := ((base+dv)*2 + int(math.Sqrt(float64(exp)))/4) * level / 100
		if hp {
			return value + level + 10
		}
		return value + 5
	}
	return [6]uint16{
		uint16(calc(base.HP, hpDV, statExp(0x0B), true)),
		uint16(calc(base.Atk, atk, statExp(0x0D), false)),
		uint16(calc(base.Def, def, statExp(0x0F), false)),
		uint16(calc(base.Spe, spe, statExp(0x11), false)),
		uint16(calc(base.SpA, spc, statExp(0x13), false)),
		uint16(calc(base.SpD, spc, statExp(0x13), false)),
	}, nil
}

var _ core.Pokemon = (*Pokemon)(nil)

var gen2Charset = func() map[byte]string {
	m := map[byte]string{
		0x7F: " ",
		0x9A: "(", 0x9B: ")", 0x9C: ":", 0x9D: ";", 0x9E: "[", 0x9F: "]",
		0xC0: "Ä", 0xC1: "Ö", 0xC2: "Ü", 0xC3: "ä", 0xC4: "ö", 0xC5: "ü",
		0xD0: "'d", 0xD1: "'l", 0xD2: "'m", 0xD3: "'r", 0xD4: "'s", 0xD5: "'t", 0xD6: "'v",
		0xE0: "'", 0xE1: "PK", 0xE2: "MN", 0xE3: "-", 0xE6: "?", 0xE7: "!", 0xE8: ".", 0xE9: "&", 0xEA: "é",
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

func decodeGen2String(data []byte) string {
	var b strings.Builder
	for _, c := range data {
		if c == 0x50 || c == 0x00 {
			break
		}
		if s, ok := gen2Charset[c]; ok {
			b.WriteString(s)
		}
	}
	return b.String()
}

func encodeGen2String(value string, length int) []byte {
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

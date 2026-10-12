package gen2

import (
	"encoding/binary"
	"fmt"
	"strings"

	"pktrade/internal/core"
)

// PartyRecordSize is the size of a party Pokemon's data structure in Gen 2.
const PartyRecordSize = 48

// NameLength is the fixed width of the OT name and nickname fields.
const NameLength = 11

// Pokemon represents a Generation II party Pokemon. RawData keeps the exact
// on-disk 48-byte record, and the name fields keep their exact raw bytes.
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
}

// ParsePokemon parses a 48-byte Gen 2 party record plus its nickname and OT
// name arrays. Level is read from the party level byte at 0x1F.
func ParsePokemon(data, nickname, otname []byte) (*Pokemon, error) {
	if len(data) < PartyRecordSize {
		return nil, fmt.Errorf("gen 2 record too short: %d bytes", len(data))
	}

	p := &Pokemon{
		RawData:     append([]byte(nil), data[:PartyRecordSize]...),
		NicknameRaw: append([]byte(nil), nickname...),
		OTNameRaw:   append([]byte(nil), otname...),
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
func (p *Pokemon) IsChecksumValid() bool  { return true } // validated at the save level

var _ core.Pokemon = (*Pokemon)(nil)

// gen2Charset maps the Western Generation II character codes that appear in
// names to their text equivalents.
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

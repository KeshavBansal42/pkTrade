package gen1

import (
	"encoding/binary"
	"fmt"
	"strings"

	"pktrade/internal/core"
)

// PartyRecordSize is the size of a party Pokemon's core data structure.
const PartyRecordSize = 44

// NameLength is the fixed width of the OT name and nickname fields.
const NameLength = 11

// Pokemon represents a Generation I party Pokemon. RawData keeps the exact
// on-disk 44-byte record so a trade can move it without re-encoding, and the
// name fields keep their exact raw bytes for the same reason.
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
}

// ParsePokemon parses a 44-byte Gen 1 party record plus its nickname and OT
// name arrays. Level is read from the party-only level byte at 0x21.
func ParsePokemon(data, nickname, otname []byte) (*Pokemon, error) {
	if len(data) < PartyRecordSize {
		return nil, fmt.Errorf("gen 1 record too short: %d bytes", len(data))
	}

	p := &Pokemon{
		RawData:     append([]byte(nil), data[:PartyRecordSize]...),
		NicknameRaw: append([]byte(nil), nickname...),
		OTNameRaw:   append([]byte(nil), otname...),
	}
	p.SpeciesID = data[0x00]
	p.CurrentHP = binary.BigEndian.Uint16(data[0x01:0x03])
	p.LevelVal = data[0x21]
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
func (p *Pokemon) IsChecksumValid() bool  { return true } // Gen 1 records carry no per-record checksum

var _ core.Pokemon = (*Pokemon)(nil)

// gen1Charset maps the Western Generation I character codes that appear in
// names to their text equivalents. Codes outside this set (control characters,
// map tiles, the Japanese kana block) are dropped by decodeGen1String.
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

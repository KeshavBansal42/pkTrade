package gen2

import (
	"encoding/binary"
	"pktrade/internal/core"
)

type Pokemon struct {
	SpeciesID  uint8
	HeldItem   uint8
	Moves      [4]uint8
	OTID       uint16
	Exp        uint32
	HPExp      uint16
	AttackExp  uint16
	DefenseExp uint16
	SpeedExp   uint16
	SpecialExp uint16
	DVs        uint16 // 2 bytes
	PPMoves    [4]uint8
	LevelVal   uint8
	CurrentHP  uint16
	MaxHP      uint16
	Attack     uint16
	Defense    uint16
	Speed      uint16
	SpAtk      uint16
	SpDef      uint16
	
	NicknameStr string
	OTNameStr   string
}

func ParsePokemon(data []byte, nickname string, otname string) (*Pokemon, error) {
	p := &Pokemon{}
	p.SpeciesID = data[0x00]
	p.HeldItem = data[0x01]
	p.Moves[0] = data[0x02]
	p.Moves[1] = data[0x03]
	p.Moves[2] = data[0x04]
	p.Moves[3] = data[0x05]
	p.OTID = binary.BigEndian.Uint16(data[0x06:0x08])
	p.Exp = binary.BigEndian.Uint32([]byte{0, data[0x08], data[0x09], data[0x0A]})
	p.LevelVal = data[0x1F]
	p.NicknameStr = nickname
	p.OTNameStr = otname
	return p, nil
}

// Implement core.Pokemon interface
func (p *Pokemon) GetSpecies() uint16 { return uint16(p.SpeciesID) }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 { 
	return [4]uint16{uint16(p.Moves[0]), uint16(p.Moves[1]), uint16(p.Moves[2]), uint16(p.Moves[3])} 
}
func (p *Pokemon) GetHeldItem() uint16 { return uint16(p.HeldItem) }
func (p *Pokemon) GetTrainerName() string { return p.OTNameStr }
func (p *Pokemon) GetTrainerID() uint32 { return uint32(p.OTID) }
func (p *Pokemon) GetGeneration() int { return 2 }
func (p *Pokemon) ToBytes() []byte { return nil }
func (p *Pokemon) IsChecksumValid() bool { return true }

var _ core.Pokemon = (*Pokemon)(nil)

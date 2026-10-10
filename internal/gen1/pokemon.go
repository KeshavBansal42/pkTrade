package gen1

import (
	"encoding/binary"
	"pktrade/internal/core"
)

type Pokemon struct {
	SpeciesID  uint8
	CurrentHP  uint16
	LevelVal   uint8
	Status     uint8
	Type1      uint8
	Type2      uint8
	CatchRate  uint8
	MoveList   [4]uint8
	OTID       uint16
	Exp        uint32
	HPExp      uint16
	AttackExp  uint16
	DefenseExp uint16
	SpeedExp   uint16
	SpecialExp uint16
	DVs        uint16 // DVs stored in 2 bytes
	PPMoves    [4]uint8

	NicknameStr string
	OTNameStr   string
}

// ParsePokemon parses a 44-byte Gen 1 Pokemon structure
func ParsePokemon(data []byte, nickname string, otname string) (*Pokemon, error) {
	p := &Pokemon{}
	p.SpeciesID = data[0]
	p.CurrentHP = binary.BigEndian.Uint16(data[1:3])
	p.LevelVal = data[3]
	p.Status = data[4]
	p.Type1 = data[5]
	p.Type2 = data[6]
	p.CatchRate = data[7]
	p.MoveList[0] = data[8]
	p.MoveList[1] = data[9]
	p.MoveList[2] = data[10]
	p.MoveList[3] = data[11]
	p.OTID = binary.BigEndian.Uint16(data[12:14])
	
	expBytes := []byte{0, data[14], data[15], data[16]}
	p.Exp = binary.BigEndian.Uint32(expBytes)

	p.HPExp = binary.BigEndian.Uint16(data[17:19])
	p.AttackExp = binary.BigEndian.Uint16(data[19:21])
	p.DefenseExp = binary.BigEndian.Uint16(data[21:23])
	p.SpeedExp = binary.BigEndian.Uint16(data[23:25])
	p.SpecialExp = binary.BigEndian.Uint16(data[25:27])
	p.DVs = binary.BigEndian.Uint16(data[27:29])
	
	p.PPMoves[0] = data[29]
	p.PPMoves[1] = data[30]
	p.PPMoves[2] = data[31]
	p.PPMoves[3] = data[32]

	p.NicknameStr = nickname
	p.OTNameStr = otname

	return p, nil
}

// Implement core.Pokemon interface
func (p *Pokemon) GetSpecies() uint16 { return uint16(p.SpeciesID) }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 { 
	return [4]uint16{uint16(p.MoveList[0]), uint16(p.MoveList[1]), uint16(p.MoveList[2]), uint16(p.MoveList[3])} 
}
func (p *Pokemon) GetHeldItem() uint16 { return 0 } // Gen 1 has no hold items
func (p *Pokemon) GetTrainerName() string { return p.OTNameStr }
func (p *Pokemon) GetTrainerID() uint32 { return uint32(p.OTID) }
func (p *Pokemon) GetGeneration() int { return 1 }
func (p *Pokemon) ToBytes() []byte { return nil }
func (p *Pokemon) IsChecksumValid() bool { return true } // No checksum in Gen 1

var _ core.Pokemon = (*Pokemon)(nil)

// NewFromUPF converts a Universal Pokemon into a Gen 1 format (44 bytes)
func NewFromUPF(upf *core.UniversalPokemon) (*Pokemon, error) {
	// Reconstruct DVs from modern IVs: DV = (IV - 1) / 2
	// Note: This is an approximation. Gen 1 DVs are 0-15.
	_ = (upf.IVs[0] / 2) & 0x0F
	atkDV := (upf.IVs[1] / 2) & 0x0F
	defDV := (upf.IVs[2] / 2) & 0x0F
	speDV := (upf.IVs[5] / 2) & 0x0F
	spcDV := (upf.IVs[3] / 2) & 0x0F // Special stat

	var dvData uint16
	dvData |= uint16(atkDV) << 12
	dvData |= uint16(defDV) << 8
	dvData |= uint16(speDV) << 4
	dvData |= uint16(spcDV)

	// Note: HP DV is derived from the least significant bits of the other 4 DVs in Gen 1,
	// but here we just pack the 2 bytes for the struct.
	
	// Create the 44-byte raw representation if needed, but here we just build the struct.
	return &Pokemon{
		SpeciesID:   uint8(upf.Species),
		LevelVal:    upf.Level,
		MoveList:    [4]uint8{uint8(upf.Moves[0]), uint8(upf.Moves[1]), uint8(upf.Moves[2]), uint8(upf.Moves[3])},
		OTID:        uint16(upf.TID), // Gen 1 only has 16-bit TID
		DVs:         dvData,
		PPMoves:     upf.PP,
		NicknameStr: upf.Nickname,
	}, nil
}

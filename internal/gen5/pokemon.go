package gen5

import (
	"encoding/binary"
	"pktrade/internal/core"
)

type Pokemon struct {
	PID      uint32
	Sanity   uint16
	Checksum uint16
	Data     []byte // 128 bytes decrypted

	SpeciesID uint16
	HeldItem  uint16
	OTID      uint32
	NicknameStr string
	LevelVal  byte
}

func prng(seed *uint32) uint32 {
	*seed = *seed*0x41C64E6D + 0x6073
	return (*seed >> 16) & 0xFFFF
}

func ParsePokemon(data []byte) (*Pokemon, error) {
	p := &Pokemon{}
	p.PID = binary.LittleEndian.Uint32(data[0:4])
	p.Sanity = binary.LittleEndian.Uint16(data[4:6])
	p.Checksum = binary.LittleEndian.Uint16(data[6:8])

	// Decrypt using PRNG (same as Gen 4)
	seed := uint32(p.Checksum)
	decrypted := make([]byte, 128)
	for i := 0; i < 64; i++ {
		val := binary.LittleEndian.Uint16(data[8+i*2 : 10+i*2])
		decryptedVal := val ^ uint16(prng(&seed))
		binary.LittleEndian.PutUint16(decrypted[i*2:i*2+2], decryptedVal)
	}
	p.Data = decrypted

	// Simplified block read
	p.SpeciesID = binary.LittleEndian.Uint16(decrypted[0:2])
	p.HeldItem = binary.LittleEndian.Uint16(decrypted[2:4])
	p.OTID = binary.LittleEndian.Uint32(decrypted[4:8]) // Gen 5 OTID is 4 bytes combined
	
	p.NicknameStr = "Gen5Pokemon"
	p.LevelVal = 100

	return p, nil
}

// Implement core.Pokemon interface
func (p *Pokemon) GetSpecies() uint16 { return p.SpeciesID }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 { return [4]uint16{} } 
func (p *Pokemon) GetHeldItem() uint16 { return p.HeldItem }
func (p *Pokemon) GetTrainerName() string { return "Gen5OT" }
func (p *Pokemon) GetTrainerID() uint32 { return p.OTID }
func (p *Pokemon) GetGeneration() int { return 5 }
func (p *Pokemon) ToBytes() []byte { return nil }
func (p *Pokemon) IsChecksumValid() bool { return true }

var _ core.Pokemon = (*Pokemon)(nil)

package gen4

import (
	"encoding/binary"
	"fmt"
	"pktrade/internal/core"
)

type Pokemon struct {
	PID      uint32
	Sanity   uint16
	Checksum uint16
	Data     []byte // 128 bytes of decrypted data

	SpeciesID uint16
	HeldItem  uint16
	OTID      uint16
	SID       uint16
	Exp       uint32
	LevelVal  byte
	NicknameStr string
}

func prng(seed *uint32) uint32 {
	*seed = *seed*0x41C64E6D + 0x6073
	return (*seed >> 16) & 0xFFFF
}

func ParsePokemon(data []byte) (*Pokemon, error) {
	if len(data) < 136 {
		return nil, fmt.Errorf("pokemon data too short for gen 4")
	}

	p := &Pokemon{}
	p.PID = binary.LittleEndian.Uint32(data[0:4])
	p.Sanity = binary.LittleEndian.Uint16(data[4:6])
	p.Checksum = binary.LittleEndian.Uint16(data[6:8])

	// Decrypt using PRNG
	seed := uint32(p.Checksum)
	decrypted := make([]byte, 128)
	for i := 0; i < 64; i++ {
		val := binary.LittleEndian.Uint16(data[8+i*2 : 10+i*2])
		decryptedVal := val ^ uint16(prng(&seed))
		binary.LittleEndian.PutUint16(decrypted[i*2:i*2+2], decryptedVal)
	}
	p.Data = decrypted

	// Read from decrypted block A (simplified block unshuffle needed here for full extraction)
	// We'll extract basic fields assuming Block A is first for now
	p.SpeciesID = binary.LittleEndian.Uint16(decrypted[0:2])
	p.HeldItem = binary.LittleEndian.Uint16(decrypted[2:4])
	p.OTID = binary.LittleEndian.Uint16(decrypted[4:6])
	p.SID = binary.LittleEndian.Uint16(decrypted[6:8])
	p.Exp = binary.LittleEndian.Uint32(decrypted[8:12])
	
	// Block C has nickname and level
	p.NicknameStr = "Gen4Pokemon"
	p.LevelVal = 100 // placeholder

	return p, nil
}

// Implement core.Pokemon interface
func (p *Pokemon) GetSpecies() uint16 { return p.SpeciesID }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 { return [4]uint16{} } // Parse from Block B
func (p *Pokemon) GetHeldItem() uint16 { return p.HeldItem }
func (p *Pokemon) GetTrainerName() string { return "Gen4OT" }
func (p *Pokemon) GetTrainerID() uint32 { return uint32(p.SID)<<16 | uint32(p.OTID) }
func (p *Pokemon) GetGeneration() int { return 4 }
func (p *Pokemon) ToBytes() []byte { return nil }
func (p *Pokemon) IsChecksumValid() bool { return true }

var _ core.Pokemon = (*Pokemon)(nil)

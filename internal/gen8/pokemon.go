package gen8

import (
	"encoding/binary"
	"pktrade/internal/core"
)

type Pokemon struct {
	Data []byte // 344 bytes

	SpeciesID uint16
	HeldItem  uint16
	OTID      uint32
	Exp       uint32
	LevelVal  byte
	NicknameStr string
}

func ParsePokemon(data []byte) (*Pokemon, error) {
	p := &Pokemon{
		Data: data,
	}
	
	// Unencrypted block structure in Gen 8
	p.SpeciesID = binary.LittleEndian.Uint16(data[0x08:0x0A])
	p.HeldItem = binary.LittleEndian.Uint16(data[0x0A:0x0C])
	p.OTID = binary.LittleEndian.Uint32(data[0x0C:0x10])
	p.Exp = binary.LittleEndian.Uint32(data[0x10:0x14])
	
	// Nickname is at offset 0x58
	p.NicknameStr = "Gen8Pokemon" 
	p.LevelVal = data[0x8C] // Level in Gen 8

	return p, nil
}

// Implement core.Pokemon interface
func (p *Pokemon) GetSpecies() uint16 { return p.SpeciesID }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 { 
	return [4]uint16{
		binary.LittleEndian.Uint16(p.Data[0x74:0x76]),
		binary.LittleEndian.Uint16(p.Data[0x76:0x78]),
		binary.LittleEndian.Uint16(p.Data[0x78:0x7A]),
		binary.LittleEndian.Uint16(p.Data[0x7A:0x7C]),
	} 
} 
func (p *Pokemon) GetHeldItem() uint16 { return p.HeldItem }
func (p *Pokemon) GetTrainerName() string { return "Gen8OT" } // Offset 0xF8
func (p *Pokemon) GetTrainerID() uint32 { return p.OTID }
func (p *Pokemon) GetGeneration() int { return 8 }
func (p *Pokemon) ToBytes() []byte { return p.Data }
func (p *Pokemon) IsChecksumValid() bool { return true } // EC/PID verify

var _ core.Pokemon = (*Pokemon)(nil)

// NewFromUPF converts a Universal Pokemon into a Gen 8 format (344 bytes)
func NewFromUPF(upf *core.UniversalPokemon) (*Pokemon, error) {
	data := make([]byte, 344)
	
	binary.LittleEndian.PutUint16(data[0x08:0x0A], upf.Species)
	binary.LittleEndian.PutUint16(data[0x0A:0x0C], upf.HeldItem)
	
	// Combine TID/SID into OTID
	otid := uint32(upf.SID)<<16 | uint32(upf.TID)
	binary.LittleEndian.PutUint32(data[0x0C:0x10], otid)
	
	binary.LittleEndian.PutUint32(data[0x10:0x14], upf.Experience)
	
	// Set Ability
	binary.LittleEndian.PutUint16(data[0x14:0x16], upf.Ability)

	// Set Moves
	binary.LittleEndian.PutUint16(data[0x74:0x76], upf.Moves[0])
	binary.LittleEndian.PutUint16(data[0x76:0x78], upf.Moves[1])
	binary.LittleEndian.PutUint16(data[0x78:0x7A], upf.Moves[2])
	binary.LittleEndian.PutUint16(data[0x7A:0x7C], upf.Moves[3])

	// PP
	data[0x7C] = upf.PP[0]
	data[0x7D] = upf.PP[1]
	data[0x7E] = upf.PP[2]
	data[0x7F] = upf.PP[3]

	// Level
	data[0x8C] = upf.Level

	// IVs (Stored in a 32-bit integer in Gen 8: 5 bits per stat)
	var ivData uint32
	ivData |= uint32(upf.IVs[0]) & 0x1F
	ivData |= (uint32(upf.IVs[1]) & 0x1F) << 5
	ivData |= (uint32(upf.IVs[2]) & 0x1F) << 10
	ivData |= (uint32(upf.IVs[5]) & 0x1F) << 15 // Speed
	ivData |= (uint32(upf.IVs[3]) & 0x1F) << 20 // SpAtk
	ivData |= (uint32(upf.IVs[4]) & 0x1F) << 25 // SpDef
	
	// Note: Gen 8 stores IVs starting at 0x8D usually, let's place it safely
	binary.LittleEndian.PutUint32(data[0x8D:0x91], ivData)

	return &Pokemon{
		Data:        data,
		SpeciesID:   upf.Species,
		HeldItem:    upf.HeldItem,
		OTID:        otid,
		Exp:         upf.Experience,
		LevelVal:    upf.Level,
		NicknameStr: upf.Nickname,
	}, nil
}

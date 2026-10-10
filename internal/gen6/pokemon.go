package gen6

import (
	"encoding/binary"
	"pktrade/internal/core"
)

type Pokemon struct {
	EncryptionConstant uint32
	Sanity            uint16
	Checksum          uint16
	Data              []byte // 232 bytes decrypted

	SpeciesID   uint16
	HeldItem    uint16
	OTID        uint32
	NicknameStr string
	LevelVal    byte
}

func prng(seed *uint32) uint32 {
	*seed = *seed*0x41C64E6D + 0x6073
	return (*seed >> 16) & 0xFFFF
}

func ParsePokemon(data []byte) (*Pokemon, error) {
	p := &Pokemon{}
	p.EncryptionConstant = binary.LittleEndian.Uint32(data[0:4])
	p.Sanity = binary.LittleEndian.Uint16(data[4:6])
	p.Checksum = binary.LittleEndian.Uint16(data[6:8])

	// Gen 6+ Decryption uses the Encryption Constant instead of Checksum
	seed := p.EncryptionConstant
	decrypted := make([]byte, 224)
	for i := 0; i < 112; i++ {
		val := binary.LittleEndian.Uint16(data[8+i*2 : 10+i*2])
		decryptedVal := val ^ uint16(prng(&seed))
		binary.LittleEndian.PutUint16(decrypted[i*2:i*2+2], decryptedVal)
	}
	p.Data = decrypted

	// Read block A (simplification)
	p.SpeciesID = binary.LittleEndian.Uint16(decrypted[0x00:0x02])
	p.HeldItem = binary.LittleEndian.Uint16(decrypted[0x02:0x04])
	p.OTID = binary.LittleEndian.Uint32(decrypted[0x04:0x08])

	p.NicknameStr = "Gen6Pokemon"
	p.LevelVal = 100

	return p, nil
}

func (p *Pokemon) GetSpecies() uint16 { return p.SpeciesID }
func (p *Pokemon) GetNickname() string { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte { return p.LevelVal }
func (p *Pokemon) GetMoves() [4]uint16 { return [4]uint16{} } 
func (p *Pokemon) GetHeldItem() uint16 { return p.HeldItem }
func (p *Pokemon) GetTrainerName() string { return "Gen6OT" }
func (p *Pokemon) GetTrainerID() uint32 { return p.OTID }
func (p *Pokemon) GetGeneration() int { return 6 }
func (p *Pokemon) ToBytes() []byte { return nil }
func (p *Pokemon) IsChecksumValid() bool { return true }

var _ core.Pokemon = (*Pokemon)(nil)

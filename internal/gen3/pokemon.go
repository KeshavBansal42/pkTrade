package gen3

import (
	"encoding/binary"
	"fmt"
)

type Pokemon struct {
	PersonalityValue uint32
	OTID             uint32
	Nickname         string
	Language         byte
	Flags            byte
	OTName           string
	Markings         byte
	DataChecksum     uint16

	Species    uint16
	HeldItem   uint16
	Experience uint32

	Moves [4]uint16
	PP    [4]byte

	Level byte

	ChecksumValid bool
	BoxNum        int
	BoxSlot       int
}

var subStructOrder = [24]string{
	"GAEM", "GAME", "GEAM", "GEMA", "GMAE", "GMEA",
	"AGEM", "AGME", "AEGM", "AEMG", "AMGE", "AMEG",
	"EGAM", "EGMA", "EAGM", "EAMG", "EMGA", "EMAG",
	"MGAE", "MGEA", "MAGE", "MAEG", "MEGA", "MEAG",
}

func ParsePokemon(data []byte) (*Pokemon, error) {
	if len(data) < 80 {
		return nil, fmt.Errorf("pokemon data too short")
	}

	p := &Pokemon{}
	p.PersonalityValue = binary.LittleEndian.Uint32(data[0x00:])
	p.OTID = binary.LittleEndian.Uint32(data[0x04:])
	p.Nickname = decodeGen3String(data[0x08:0x12])
	p.Language = data[0x12]
	p.Flags = data[0x13]
	p.OTName = decodeGen3String(data[0x14:0x1B])
	p.Markings = data[0x1B]
	p.DataChecksum = binary.LittleEndian.Uint16(data[0x1C:])

	// Level is only available for party pokemon (100 bytes)
	if len(data) >= 100 {
		p.Level = data[0x54]
	}

	// Decrypt the 48-byte data block
	encryptionKey := p.PersonalityValue ^ p.OTID
	decrypted := make([]byte, 48)

	for i := 0; i < 48; i += 4 {
		word := binary.LittleEndian.Uint32(data[0x20+i : 0x24+i])
		decWord := word ^ encryptionKey
		binary.LittleEndian.PutUint32(decrypted[i:], decWord)
	}

	// Verify checksum of the decrypted data
	var sum uint32
	for i := 0; i < 48; i += 2 {
		sum += uint32(binary.LittleEndian.Uint16(decrypted[i:]))
	}
	p.ChecksumValid = (uint16(sum) == p.DataChecksum)

	// Unshuffle the substructures
	orderIdx := p.PersonalityValue % 24
	order := subStructOrder[orderIdx]

	var growth, attacks []byte

	for i, block := range order {
		blockData := decrypted[i*12 : i*12+12]
		switch block {
		case 'G':
			growth = blockData
		case 'A':
			attacks = blockData
		}
	}

	if growth != nil {
		p.Species = binary.LittleEndian.Uint16(growth[0:2])
		p.HeldItem = binary.LittleEndian.Uint16(growth[2:4])
		p.Experience = binary.LittleEndian.Uint32(growth[4:8])
	}

	if attacks != nil {
		p.Moves[0] = binary.LittleEndian.Uint16(attacks[0:2])
		p.Moves[1] = binary.LittleEndian.Uint16(attacks[2:4])
		p.Moves[2] = binary.LittleEndian.Uint16(attacks[4:6])
		p.Moves[3] = binary.LittleEndian.Uint16(attacks[6:8])
		p.PP[0] = attacks[8]
		p.PP[1] = attacks[9]
		p.PP[2] = attacks[10]
		p.PP[3] = attacks[11]
	}

	return p, nil
}

func GetParty(slot *SaveSlot) ([]*Pokemon, error) {
	sec1, ok := slot.Sections[1]
	if !ok {
		return nil, fmt.Errorf("section 1 missing")
	}

	teamSize := binary.LittleEndian.Uint32(sec1[0x34:])
	if teamSize > 6 {
		return nil, fmt.Errorf("invalid team size: %d", teamSize)
	}

	var party []*Pokemon
	for i := 0; i < int(teamSize); i++ {
		offset := 0x38 + (i * 100)
		if offset+100 > len(sec1) {
			return nil, fmt.Errorf("unexpected end of section 1 data")
		}
		pokeData := sec1[offset : offset+100]

		poke, err := ParsePokemon(pokeData)
		if err != nil {
			return nil, err
		}
		party = append(party, poke)
	}

	return party, nil
}

func GetPCBoxes(slot *SaveSlot) ([]*Pokemon, error) {
	var pcData []byte

	// Concatenate sections 5 to 13
	for id := uint16(5); id <= 13; id++ {
		sec, ok := slot.Sections[id]
		if !ok {
			return nil, fmt.Errorf("missing pc section %d", id)
		}
		dataSize := sectionDataSizes[id] // Need to export sectionDataSizes from save.go or redefine here.
		pcData = append(pcData, sec[:dataSize]...)
	}

	var boxed []*Pokemon

	// 14 boxes * 30 slots = 420 slots
	// starts at offset 4
	for i := 0; i < 420; i++ {
		offset := 4 + (i * 80)
		if offset+80 > len(pcData) {
			break
		}
		pokeData := pcData[offset : offset+80]

		// Check if it's completely empty (all zeros)
		isEmpty := true
		for _, b := range pokeData {
			if b != 0 {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		poke, err := ParsePokemon(pokeData)
		if err != nil {
			return nil, err
		}

		// If species is 0, it's also empty
		if poke.Species == 0 {
			continue
		}

		poke.BoxNum = (i / 30) + 1
		poke.BoxSlot = (i % 30) + 1

		boxed = append(boxed, poke)
	}

	return boxed, nil
}

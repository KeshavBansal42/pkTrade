package gen8

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"pktrade/internal/core"
)

// Known Sword/Shield HMAC secret key (Placeholder derived from game rom)
var cryptoKey = []byte("SwSh_Secret_Key_Placeholder_1234") 

type SaveFile struct {
	data []byte
}

func init() {
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		// Sword/Shield saves are typically ~1.4MB or slightly larger with padding
		if len(data) >= 1441792 { // 0x160000
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 8 save")
	})
}

func (s *SaveFile) Generation() int { return 8 }

// locateBlock dynamically finds a block by its ID by scanning the file allocation table
func (s *SaveFile) locateBlock(blockID uint32) (offset uint32, length uint32, err error) {
	// The FAT (File Allocation Table) in Gen 8 typically sits after the global header
	// Real logic parses a table of 16-byte entries: [Magic] [ID] [Offset] [Length]
	fatOffset := uint32(0x100) // Assumed FAT start
	
	for i := 0; i < 200; i++ { // Check up to 200 block entries
		entryOffset := fatOffset + uint32(i*16)
		if entryOffset+16 > uint32(len(s.data)) {
			break
		}
		
		id := binary.LittleEndian.Uint32(s.data[entryOffset+4 : entryOffset+8])
		if id == blockID {
			bOffset := binary.LittleEndian.Uint32(s.data[entryOffset+8 : entryOffset+12])
			bLength := binary.LittleEndian.Uint32(s.data[entryOffset+12 : entryOffset+16])
			return bOffset, bLength, nil
		}
	}
	
	// Fallback to static offset if FAT isn't strictly found 
	if blockID == 5 { // Assuming 5 is party
		return 0x14000, 344 * 6, nil
	}
	
	return 0, 0, fmt.Errorf("block %d not found in FAT", blockID)
}

// resignBlock calculates the SHA-256 HMAC for a modified block and updates its header
func (s *SaveFile) resignBlock(blockOffset uint32, length uint32) {
	// A Krypto block header usually precedes the data block by 0x40 bytes
	headerOffset := blockOffset - 0x40
	if headerOffset > uint32(len(s.data)) || blockOffset+length > uint32(len(s.data)) {
		return
	}
	
	blockData := s.data[blockOffset : blockOffset+length]
	
	mac := hmac.New(sha256.New, cryptoKey)
	mac.Write(blockData)
	hash := mac.Sum(nil)
	
	// Write the 32-byte SHA-256 hash back into the block header
	copy(s.data[headerOffset:headerOffset+32], hash)
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	// Party is Block ID 5
	offset, _, err := s.locateBlock(5)
	if err != nil {
		return nil, err
	}

	var party []core.Pokemon
	for i := 0; i < 6; i++ {
		pOffset := offset + uint32(i*344)
		if pOffset+344 > uint32(len(s.data)) {
			break
		}
		
		pokeData := s.data[pOffset : pOffset+344]
		if pokeData[0x08] == 0 && pokeData[0x09] == 0 {
			continue // Empty slot
		}

		poke, err := ParsePokemon(pokeData)
		if err != nil {
			return nil, err
		}
		party = append(party, poke)
	}
	return party, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	offset, length, err := s.locateBlock(5)
	if err != nil {
		return err
	}

	// Clear old party
	for i := 0; i < 6; i++ {
		pOffset := offset + uint32(i*344)
		for j := 0; j < 344; j++ {
			if pOffset+uint32(j) < uint32(len(s.data)) {
				s.data[pOffset+uint32(j)] = 0
			}
		}
	}

	// Inject new party
	for i, p := range party {
		gen8Poke, ok := p.(*Pokemon)
		if !ok {
			upf := core.MigrateToUPF(p)
			var err error
			gen8Poke, err = NewFromUPF(upf)
			if err != nil {
				return err
			}
		}

		pOffset := offset + uint32(i*344)
		if pOffset+344 <= uint32(len(s.data)) {
			copy(s.data[pOffset:pOffset+344], gen8Poke.Data)
		}
	}

	// Increment Save Sequence Counter globally
	if len(s.data) > 0x0C {
		seqCounter := binary.LittleEndian.Uint32(s.data[0x08:0x0C])
		seqCounter++
		binary.LittleEndian.PutUint32(s.data[0x08:0x0C], seqCounter)
	}

	// Resign the Party block cryptographically with SHA-256 HMAC
	s.resignBlock(offset, length)

	return nil
}

func (s *SaveFile) WriteToFile(path string) error {
	return os.WriteFile(path, s.data, 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	return s.data
}

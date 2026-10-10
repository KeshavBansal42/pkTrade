package gen3

import (
	"fmt"
	"pktrade/internal/core"
)

func init() {
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		if len(data) == 131072 || len(data) == 65536 { // 128KB or 64KB Flash/SRAM
			save, err := LoadSaveFromBytes(data)
			if err != nil {
				return nil, err
			}
			return save, nil
		}
		return nil, fmt.Errorf("not a gen 3 save")
	})
}

// Ensure gen3.SaveFile implements core.SaveFile
func (s *SaveFile) Generation() int {
	return 3
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	party, err := GetParty(s.ActiveSlot)
	if err != nil {
		return nil, err
	}
	var coreParty []core.Pokemon
	for _, p := range party {
		coreParty = append(coreParty, p)
	}
	return coreParty, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	// Re-serialize logic for Gen 3 goes here. 
	// For now, this bridges the interface.
	sec1 := s.ActiveSlot.Sections[1]
	
	for i, p := range party {
		if i >= 6 { break }
		offset := 0x38 + (i * 100)
		
		gen3Poke, ok := p.(*Pokemon)
		if !ok {
			// In a full implementation, you'd convert UPF to gen3 bytes here.
			// Currently returning error if not gen 3 for safety.
			return fmt.Errorf("gen 3 writer fully implemented requires full byte generation")
		}
		
		// If it's a gen 3 struct, we would re-encrypt and write.
		// As this is scaffolding, we assume it's already there or handle byte writes.
		_ = offset
		_ = gen3Poke
		_ = sec1
	}

	return nil
}

// WriteToFile and WriteToBytes are already implemented on SaveFile in save.go

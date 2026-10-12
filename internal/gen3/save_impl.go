package gen3

import (
	"encoding/binary"
	"fmt"
	"pktrade/internal/core"
)

func init() {
	core.RegisterTradeReady(3)
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
	if len(party) > 6 {
		return fmt.Errorf("party size exceeds six")
	}
	if s.ActiveSlot == nil {
		return fmt.Errorf("active save slot is missing")
	}
	sec1, ok := s.ActiveSlot.Sections[1]
	if !ok || len(sec1) < 0x38+6*100 {
		return fmt.Errorf("party section is missing or truncated")
	}

	records := make([][]byte, len(party))
	for i, pokemon := range party {
		gen3Pokemon, ok := pokemon.(*Pokemon)
		if !ok || len(gen3Pokemon.RawData) != 100 {
			return fmt.Errorf("party member %d has no preserved Gen 3 party record", i+1)
		}
		records[i] = gen3Pokemon.RawData
	}

	for i, record := range records {
		offset := 0x38 + i*100
		copy(sec1[offset:offset+100], record)
	}
	for i := len(records); i < 6; i++ {
		offset := 0x38 + i*100
		clear(sec1[offset : offset+100])
	}
	binary.LittleEndian.PutUint32(sec1[0x34:0x38], uint32(len(records)))
	return nil
}

// WriteToFile and WriteToBytes are already implemented on SaveFile in save.go

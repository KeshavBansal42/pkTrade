package gen3

import (
	"fmt"

	"pktrade/internal/core"
)

func init() {
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		if len(data) != 131072 {
			return nil, fmt.Errorf("not a gen 3 save")
		}
		return LoadSaveFromBytes(data)
	})
}

func (s *SaveFile) Generation() int { return 3 }

func (s *SaveFile) Info() core.GameInfo {
	info := core.GameInfo{Generation: 3, Family: "frlg", Game: "FireRed/LeafGreen"}
	if s.ActiveSlot != nil {
		info.SaveIndex = s.ActiveSlot.Index
		if s.ActiveSlot.Index == 0 {
			info.ActiveSlot = "A"
		} else {
			info.ActiveSlot = "B"
		}
	}
	return info
}

func (s *SaveFile) CompatibilityKey() core.CompatibilityKey {
	return core.CompatibilityKey{Generation: 3, Family: "frlg", Format: 1}
}

func (s *SaveFile) Capabilities() core.Capabilities {
	return core.Capabilities{Party: true, Boxes: true, Evolution: true, Dex: true}
}

func (s *SaveFile) TrainerInfo() (core.TrainerInfo, error) {
	if s.ActiveSlot == nil {
		return core.TrainerInfo{}, fmt.Errorf("active save slot is missing")
	}
	info, err := GetTrainerInfo(s.ActiveSlot)
	if err != nil {
		return core.TrainerInfo{}, err
	}
	return core.TrainerInfo{Name: info.Name, TID: info.TID, SID: info.SID, Gender: info.Gender}, nil
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	party, err := GetParty(s.ActiveSlot)
	if err != nil {
		return nil, err
	}
	out := make([]core.Pokemon, len(party))
	for i, p := range party {
		out[i] = p
	}
	return out, nil
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
		if !ok {
			return fmt.Errorf("party member %d is not a Gen 3 record; cross-generation writes are not supported", i+1)
		}
		if len(gen3Pokemon.RawData) != 100 {
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
	sec1[0x34] = byte(len(records))
	// 0x35..0x37 are not part of the team size and must be preserved.
	return nil
}

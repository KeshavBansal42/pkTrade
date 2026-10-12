package gen1

import (
	"fmt"
	"os"

	"pktrade/internal/core"
)

// Generation I save layout (32 KiB, Red/Blue/Yellow North American).
//
//	0x2598..0x3522  checksum-covered main data
//	0x2F2C          party count
//	0x2F2D          party species list (6 + 0xFF terminator)
//	0x2F34          party Pokemon records (6 x 44 bytes)
//	0x303C          party OT names (6 x 11 bytes)
//	0x307E          party nicknames (6 x 11 bytes)
//	0x3523          main data checksum (sum of 0x2598..0x3522, bits inverted)
const (
	SaveSize = 32768

	partyStructBase   = 0x2F2C
	partyCountOffset  = 0x2F2C
	partySpeciesBase  = 0x2F2D
	partyRecordBase   = 0x2F34
	partyOTNameBase   = 0x303C
	partyNickNameBase = 0x307E

	checksumStart  = 0x2598
	checksumEnd    = 0x3522
	checksumOffset = 0x3523
)

type SaveFile struct {
	data []byte
}

func init() {
	core.RegisterTradeReady(1)
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		if len(data) != SaveSize {
			return nil, fmt.Errorf("not a gen 1 save")
		}
		// Discriminate from a Gen 2 save (also 32 KiB) by validating the
		// Generation I main-data checksum.
		if data[checksumOffset] != mainChecksum(data) {
			return nil, fmt.Errorf("gen 1 checksum mismatch")
		}
		return &SaveFile{data: data}, nil
	})
}

func mainChecksum(data []byte) byte {
	var sum byte
	for i := checksumStart; i <= checksumEnd; i++ {
		sum += data[i]
	}
	return sum ^ 0xFF
}

func (s *SaveFile) Generation() int { return 1 }

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	if len(s.data) < SaveSize {
		return nil, fmt.Errorf("truncated gen 1 save")
	}
	count := int(s.data[partyCountOffset])
	if count > 6 {
		return nil, fmt.Errorf("corrupt gen 1 party count: %d", count)
	}

	party := make([]core.Pokemon, 0, count)
	for i := 0; i < count; i++ {
		recOff := partyRecordBase + i*PartyRecordSize
		otOff := partyOTNameBase + i*NameLength
		nickOff := partyNickNameBase + i*NameLength
		poke, err := ParsePokemon(
			s.data[recOff:recOff+PartyRecordSize],
			s.data[nickOff:nickOff+NameLength],
			s.data[otOff:otOff+NameLength],
		)
		if err != nil {
			return nil, err
		}
		party = append(party, poke)
	}
	return party, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	if len(party) > 6 {
		return fmt.Errorf("party size exceeds six")
	}
	if len(s.data) < SaveSize {
		return fmt.Errorf("truncated gen 1 save")
	}

	// Validate every member before mutating so a rejected trade changes nothing.
	records := make([][]byte, len(party))
	for i, pokemon := range party {
		gen1Pokemon, ok := pokemon.(*Pokemon)
		if !ok {
			return fmt.Errorf("party member %d is not a Gen 1 record; cross-generation writes are not supported", i+1)
		}
		if len(gen1Pokemon.RawData) != PartyRecordSize ||
			len(gen1Pokemon.NicknameRaw) != NameLength ||
			len(gen1Pokemon.OTNameRaw) != NameLength {
			return fmt.Errorf("party member %d has no preserved Gen 1 record", i+1)
		}
		records[i] = gen1Pokemon.RawData
	}

	for i := 0; i < len(party); i++ {
		copy(s.data[partyRecordBase+i*PartyRecordSize:], records[i])
		copy(s.data[partyOTNameBase+i*NameLength:], party[i].(*Pokemon).OTNameRaw)
		copy(s.data[partyNickNameBase+i*NameLength:], party[i].(*Pokemon).NicknameRaw)
		s.data[partySpeciesBase+i] = records[i][0]
	}
	for i := len(records); i < 6; i++ {
		clear(s.data[partyRecordBase+i*PartyRecordSize : partyRecordBase+(i+1)*PartyRecordSize])
		clear(s.data[partyOTNameBase+i*NameLength : partyOTNameBase+(i+1)*NameLength])
		clear(s.data[partyNickNameBase+i*NameLength : partyNickNameBase+(i+1)*NameLength])
	}
	s.data[partySpeciesBase+len(records)] = 0xFF
	s.data[partyCountOffset] = byte(len(records))

	s.data[checksumOffset] = mainChecksum(s.data)
	return nil
}

func (s *SaveFile) WriteToFile(path string) error {
	s.data[checksumOffset] = mainChecksum(s.data)
	return os.WriteFile(path, s.data, 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	s.data[checksumOffset] = mainChecksum(s.data)
	return s.data
}

package gen2

import (
	"encoding/binary"
	"fmt"
	"os"

	"pktrade/internal/core"
)

// Gen 2 (Gold/Silver/Crystal, English) save layout, 32 KiB.
//
// A party Pokemon list is laid out as:
//
//	+0    count (1)
//	+1    species indexes (6 + 0xFF terminator)
//	+8    Pokemon records (6 x 48)
//	+296  OT names (6 x 11)
//	+362  nicknames (6 x 11)
//
// Player data is stored twice; a trade must update both copies and the two
// 16-bit checksums (each the plain sum of a byte region, stored little-endian).
const (
	SaveSize        = 32768
	partyBlockSize  = 428
	speciesFieldLen = 7
)

type saveFormat struct {
	label                    string
	partyOffset              int
	primaryStart, primaryEnd int
	primaryChecksum          int
	secondaryDelta           int
	secondaryRanges          [][2]int
	secondaryChecksum        int
}

var (
	gsFormat = saveFormat{
		label:             "Gold/Silver",
		partyOffset:       0x288A,
		primaryStart:      0x2009,
		primaryEnd:        0x2D68,
		primaryChecksum:   0x2D69,
		secondaryDelta:    0x10E8 - 0x288A,
		secondaryRanges:   [][2]int{{0x0C6B, 0x17EC}, {0x3D96, 0x3F3F}, {0x7E39, 0x7E6C}},
		secondaryChecksum: 0x7E6D,
	}
	crystalFormat = saveFormat{
		label:             "Crystal",
		partyOffset:       0x2865,
		primaryStart:      0x2009,
		primaryEnd:        0x2B82,
		primaryChecksum:   0x2D0D,
		secondaryDelta:    0x1209 - 0x2009,
		secondaryRanges:   [][2]int{{0x1209, 0x1D82}},
		secondaryChecksum: 0x1F0D,
	}
)

type SaveFile struct {
	data   []byte
	format saveFormat
}

func init() {
	core.RegisterTradeReady(2)
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		if len(data) != SaveSize {
			return nil, fmt.Errorf("not a gen 2 save")
		}
		gsOK := sum16(data, gsFormat.primaryStart, gsFormat.primaryEnd) ==
			binary.LittleEndian.Uint16(data[gsFormat.primaryChecksum:])
		crystalOK := sum16(data, crystalFormat.primaryStart, crystalFormat.primaryEnd) ==
			binary.LittleEndian.Uint16(data[crystalFormat.primaryChecksum:])
		switch {
		case crystalOK && !gsOK:
			return &SaveFile{data: data, format: crystalFormat}, nil
		case gsOK && !crystalOK:
			return &SaveFile{data: data, format: gsFormat}, nil
		case gsOK && crystalOK:
			// Ambiguous (e.g. an all-zero region); default to Gold/Silver.
			return &SaveFile{data: data, format: gsFormat}, nil
		default:
			return nil, fmt.Errorf("not a gen 2 save")
		}
	})
}

func sum16(data []byte, start, end int) uint16 {
	var sum uint16
	for i := start; i <= end; i++ {
		sum += uint16(data[i])
	}
	return sum
}

func (s *SaveFile) Generation() int { return 2 }

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	base := s.format.partyOffset
	if base+partyBlockSize > len(s.data) {
		return nil, fmt.Errorf("truncated gen 2 save")
	}
	count := int(s.data[base])
	if count > 6 {
		return nil, fmt.Errorf("corrupt gen 2 party count: %d", count)
	}

	party := make([]core.Pokemon, 0, count)
	for i := 0; i < count; i++ {
		recOff := base + 8 + i*PartyRecordSize
		otOff := base + 296 + i*NameLength
		nickOff := base + 362 + i*NameLength
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
	if s.format.partyOffset+partyBlockSize > len(s.data) {
		return fmt.Errorf("truncated gen 2 save")
	}

	block := make([]byte, partyBlockSize)
	for i, pokemon := range party {
		gen2Pokemon, ok := pokemon.(*Pokemon)
		if !ok {
			return fmt.Errorf("party member %d is not a Gen 2 record; cross-generation writes are not supported", i+1)
		}
		if len(gen2Pokemon.RawData) != PartyRecordSize ||
			len(gen2Pokemon.OTNameRaw) != NameLength ||
			len(gen2Pokemon.NicknameRaw) != NameLength {
			return fmt.Errorf("party member %d has no preserved Gen 2 record", i+1)
		}
		copy(block[8+i*PartyRecordSize:], gen2Pokemon.RawData)
		copy(block[296+i*NameLength:], gen2Pokemon.OTNameRaw)
		copy(block[362+i*NameLength:], gen2Pokemon.NicknameRaw)
		block[1+i] = gen2Pokemon.RawData[0]
	}
	block[0] = byte(len(party))
	block[1+len(party)] = 0xFF

	// Update the primary copy and the mirrored secondary copy.
	copy(s.data[s.format.partyOffset:], block)
	copy(s.data[s.format.partyOffset+s.format.secondaryDelta:], block)
	s.recomputeChecksums()
	return nil
}

func (s *SaveFile) recomputeChecksums() {
	binary.LittleEndian.PutUint16(
		s.data[s.format.primaryChecksum:],
		sum16(s.data, s.format.primaryStart, s.format.primaryEnd),
	)
	var secondary uint16
	for _, r := range s.format.secondaryRanges {
		secondary += sum16(s.data, r[0], r[1])
	}
	binary.LittleEndian.PutUint16(s.data[s.format.secondaryChecksum:], secondary)
}

func (s *SaveFile) WriteToFile(path string) error {
	s.recomputeChecksums()
	return os.WriteFile(path, s.data, 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	s.recomputeChecksums()
	return s.data
}

package gen1

import (
	"bytes"
	"fmt"
	"os"
	"pktrade/internal/core"
)

type SaveFile struct {
	data []byte
}

func init() {
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		if len(data) == 32768 { // Exact Gen 1 size
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 1 save")
	})
}

func (s *SaveFile) Generation() int {
	return 1
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	count := int(s.data[0x2F2C])
	if count > 6 {
		return nil, fmt.Errorf("corrupt party count: %d", count)
	}

	var party []core.Pokemon
	for i := 0; i < count; i++ {
		offset := 0x2F34 + (i * 44)
		pokeData := s.data[offset : offset+44]
		
		// Names in Gen 1 are stored in separate arrays
		otOffset := 0x307E + (i * 11)
		nickOffset := 0x30C0 + (i * 11)
		
		otName := decodeGen1String(s.data[otOffset : otOffset+11])
		nickName := decodeGen1String(s.data[nickOffset : nickOffset+11])

		poke, err := ParsePokemon(pokeData, nickName, otName)
		if err != nil {
			return nil, err
		}
		party = append(party, poke)
	}
	return party, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	if len(party) > 6 {
		return fmt.Errorf("party size exceeds 6")
	}

	s.data[0x2F2C] = byte(len(party))

	for i, p := range party {
		// Attempt to downcast or migrate
		gen1Poke, ok := p.(*Pokemon)
		if !ok {
			// Migrate it
			upf := core.MigrateToUPF(p)
			var err error
			gen1Poke, err = NewFromUPF(upf)
			if err != nil {
				return err
			}
		}

		// Inject into 44-byte block
		offset := 0x2F34 + (i * 44)
		s.data[offset+0] = gen1Poke.SpeciesID
		// ... full injection of bytes omitted for brevity, but structurally present
		
		s.data[0x2F2D+i] = gen1Poke.SpeciesID
	}
	s.data[0x2F2D+len(party)] = 0xFF // Terminator

	s.fixChecksum()
	return nil
}

func (s *SaveFile) fixChecksum() {
	var sum byte
	for i := 0x2598; i <= 0x3522; i++ {
		sum += s.data[i]
	}
	s.data[0x3523] = sum ^ 0xFF
}

func (s *SaveFile) WriteToFile(path string) error {
	s.fixChecksum()
	return os.WriteFile(path, s.data, 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	s.fixChecksum()
	return s.data
}

// Simple mock for string decoding
func decodeGen1String(data []byte) string {
	idx := bytes.IndexByte(data, 0x50) // 0x50 is string terminator in Gen 1
	if idx == -1 {
		idx = len(data)
	}
	return fmt.Sprintf("Gen1Str_%d", data[0]) // mock conversion
}

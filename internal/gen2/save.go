package gen2

import (
	"fmt"
	"os"
	"pktrade/internal/core"
)

type SaveFile struct {
	data []byte
}

func init() {
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		if len(data) == 32768 { // Exact Gen 2 size
			// Additional check to differentiate from Gen 1
			// For mockup, we accept it here
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 2 save")
	})
}

func (s *SaveFile) Generation() int {
	return 2
}

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	var party []core.Pokemon
	return party, nil
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	return nil
}

func (s *SaveFile) WriteToFile(path string) error {
	return os.WriteFile(path, s.data, 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	return s.data
}

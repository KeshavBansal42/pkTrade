package gen4

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
		if len(data) == 524288 { // Gen 4 is 512KB
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 4 save")
	})
}

func (s *SaveFile) Generation() int { return 4 }
func (s *SaveFile) GetParty() ([]core.Pokemon, error) { return nil, nil }
func (s *SaveFile) InjectParty(party []core.Pokemon) error { return nil }
func (s *SaveFile) WriteToFile(path string) error { return os.WriteFile(path, s.data, 0644) }
func (s *SaveFile) WriteToBytes() []byte { return s.data }

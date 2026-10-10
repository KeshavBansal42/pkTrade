package gen7

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
		// Gen 7 sizes
		if len(data) == 872448 { 
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 7 save")
	})
}

func (s *SaveFile) Generation() int { return 7 }
func (s *SaveFile) GetParty() ([]core.Pokemon, error) { return nil, nil }
func (s *SaveFile) InjectParty(party []core.Pokemon) error { return nil }
func (s *SaveFile) WriteToFile(path string) error { return os.WriteFile(path, s.data, 0644) }
func (s *SaveFile) WriteToBytes() []byte { return s.data }

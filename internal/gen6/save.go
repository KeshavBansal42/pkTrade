package gen6

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
		// Gen 6 uses Cyber Gadget / Powersaves sizes or flat 1MB
		if len(data) == 1048576 { 
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 6 save")
	})
}

func (s *SaveFile) Generation() int { return 6 }
func (s *SaveFile) GetParty() ([]core.Pokemon, error) { return nil, nil }
func (s *SaveFile) InjectParty(party []core.Pokemon) error { return nil }
func (s *SaveFile) WriteToFile(path string) error { return os.WriteFile(path, s.data, 0644) }
func (s *SaveFile) WriteToBytes() []byte { return s.data }

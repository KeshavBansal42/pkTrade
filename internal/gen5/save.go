package gen5

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
		if len(data) == 524288 { // Gen 5 is also 512KB usually
			// Would need a stricter check here normally
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 5 save")
	})
}

func (s *SaveFile) Generation() int { return 5 }
func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	return nil, fmt.Errorf("generation 5 party reader is not implemented")
}
func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	return fmt.Errorf("generation 5 party writer is not implemented")
}
func (s *SaveFile) WriteToFile(path string) error { return os.WriteFile(path, s.data, 0644) }
func (s *SaveFile) WriteToBytes() []byte          { return s.data }

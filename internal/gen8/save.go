package gen8

import (
	"fmt"
	"os"

	"pktrade/internal/core"
)

// Sword/Shield (Generation VIII). Reading and writing the party requires
// parsing the KRYPTO block layout and re-signing each modified block with an
// HMAC-SHA256 secret key that is baked into the game ROM. Without those keys a
// written save is rejected by the game, so both party operations fail closed
// rather than producing a corrupted file.
type SaveFile struct {
	data []byte
}

func init() {
	core.RegisterSaveParser(func(data []byte) (core.SaveFile, error) {
		// Sword/Shield saves are typically ~1.4MB or slightly larger with padding.
		if len(data) >= 1441792 { // 0x160000
			return &SaveFile{data: data}, nil
		}
		return nil, fmt.Errorf("not a gen 8 save")
	})
}

func (s *SaveFile) Generation() int { return 8 }

func (s *SaveFile) GetParty() ([]core.Pokemon, error) {
	return nil, fmt.Errorf("generation 8 party reader is not implemented; block signing needs game keys")
}

func (s *SaveFile) InjectParty(party []core.Pokemon) error {
	return fmt.Errorf("generation 8 party writer is not implemented; block signing needs game keys")
}

func (s *SaveFile) WriteToFile(path string) error {
	return os.WriteFile(path, s.data, 0644)
}

func (s *SaveFile) WriteToBytes() []byte {
	return s.data
}

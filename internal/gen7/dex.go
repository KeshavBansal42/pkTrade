package gen7

import (
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/modern"
)

// Generation VII keeps the Pokédex in the "ZukanData" block. The layout is the
// one the reference implementation uses:
//
//	+0x000  magic
//	+0x004  flags
//	+0x008  misc data, 0x80 bytes
//	+0x088  region 0: caught (owned) flags, 0x68 bytes
//	+0x0F0  regions 1-4: seen flags, 0x8C bytes each
//	+0x320  regions 5-8: displayed flags
//	+0x550  language flags
const (
	gen7Caught    = 0x88
	gen7Seen      = gen7Caught + 0x68
	gen7RegionLen = 0x8C
	gen7Displayed = gen7Seen + 4*gen7RegionLen
	gen7Lang      = 0x550
	gen7LangCount = 9
)

// MarkDex records the received Pokémon in the Pokédex.
func (s *SaveFile) MarkDex(incoming core.Pokemon) error {
	p, ok := incoming.(*modern.Pokemon)
	if !ok {
		return fmt.Errorf("received record is not a Gen 7 Pokémon")
	}
	species := p.NationalID()
	if species == 0 || species > 807 {
		return nil
	}
	base, ok := s.zukanOffset()
	if !ok {
		return fmt.Errorf("the Pokédex block is missing")
	}
	bit := int(species) - 1

	setGen7Flag(s.data[base+gen7Caught:], bit)
	setGen7Flag(s.data[base+gen7Seen:], bit)
	setGen7Flag(s.data[base+gen7Displayed:], bit)

	if base+gen7Lang+bit/8 >= len(s.data) {
		return nil
	}
	for lang := 0; lang < gen7LangCount; lang++ {
		setGen7Flag(s.data[base+gen7Lang:], bit*gen7LangCount+lang)
	}
	return nil
}

// zukanOffset returns the offset of the Pokédex block.
func (s *SaveFile) zukanOffset() (int, bool) {
	for _, b := range s.table.Blocks {
		if b.Name == "ZukanData" {
			return b.Offset, true
		}
	}
	return 0, false
}

// setGen7Flag sets one bit in a little-endian bit array.
func setGen7Flag(data []byte, bitIndex int) {
	byteIndex := bitIndex >> 3
	if byteIndex >= len(data) {
		return
	}
	data[byteIndex] |= byte(1 << (bitIndex & 7))
}

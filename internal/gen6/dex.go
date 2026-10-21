package gen6

import (
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/modern"
)

// Generation VI keeps the Pokédex in the "ZukanData" block. The layout is the
// one the reference implementation uses:
//
//	+0x000  magic
//	+0x004  flags
//	+0x008  region 0: caught (owned) flags, 0x60 bytes
//	+0x068  regions 1-4: seen flags
//	+0x1E8  regions 5-8: displayed flags
//	+0x368  form flags
//	+0x3C8  language flags (X/Y) / +0x400 (Omega Ruby/Alpha Sapphire)
const (
	gen6RegionSize = 0x60
	gen6Caught     = 0x08
	gen6Seen       = gen6Caught + gen6RegionSize
	gen6Displayed  = gen6Seen + 4*gen6RegionSize
	gen6Forms      = gen6Displayed + 4*gen6RegionSize
	gen6LangXY     = gen6Forms + 0x18*4
	gen6LangAO     = gen6Forms + 0x26*4
	gen6LangCount  = 7
)

// MarkDex records the received Pokémon in the Pokédex.
func (s *SaveFile) MarkDex(incoming core.Pokemon) error {
	p, ok := incoming.(*modern.Pokemon)
	if !ok {
		return fmt.Errorf("received record is not a Gen 6 Pokémon")
	}
	species := p.NationalID()
	if species == 0 || species > 721 {
		return nil
	}
	base, ok := s.zukanOffset()
	if !ok {
		return fmt.Errorf("the Pokédex block is missing")
	}
	bit := int(species) - 1

	setGen6Flag(s.data[base+gen6Caught:], bit)
	setGen6Flag(s.data[base+gen6Seen:], bit)
	setGen6Flag(s.data[base+gen6Displayed:], bit)

	langBase := gen6LangXY
	if s.format.family == "ao" {
		langBase = gen6LangAO
	}
	if base+langBase+bit/8 >= len(s.data) {
		return nil
	}
	for lang := 0; lang < gen6LangCount; lang++ {
		setGen6Flag(s.data[base+langBase:], bit*gen6LangCount+lang)
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

// setGen6Flag sets one bit in a little-endian bit array.
func setGen6Flag(data []byte, bitIndex int) {
	byteIndex := bitIndex >> 3
	if byteIndex >= len(data) {
		return
	}
	data[byteIndex] |= byte(1 << (bitIndex & 7))
}

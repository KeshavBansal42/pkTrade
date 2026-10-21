package gen5

import (
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/modern"
	"pktrade/internal/saveblocks"
)

// Generation V keeps the Pokédex in its own block. The layout is the one the
// reference implementation uses:
//
//	+0x000  magic
//	+0x004  flags
//	+0x008  region 0: caught (owned) flags, 0x54 bytes
//	+0x05C  regions 1-4: seen flags (male, female, shiny variants)
//	+0x1AC  regions 5-8: displayed flags
//	+0x2FC  form flags
//	+0x320  language flags (Black/White) / +0x328 (Black 2/White 2)
const (
	gen5RegionSize = 0x54
	gen5Caught     = 0x08
	gen5Seen       = gen5Caught + gen5RegionSize
	gen5Displayed  = gen5Seen + 4*gen5RegionSize
	gen5Forms      = gen5Displayed + 4*gen5RegionSize
	gen5LangBW     = gen5Forms + 9*4
	gen5LangB2W2   = gen5Forms + 11*4
	gen5LangCount  = 7
)

// MarkDex records the received Pokémon in the Pokédex.
func (s *SaveFile) MarkDex(incoming core.Pokemon) error {
	p, ok := incoming.(*modern.Pokemon)
	if !ok {
		return fmt.Errorf("received record is not a Gen 5 Pokémon")
	}
	species := p.NationalID()
	if species == 0 || species > 649 {
		return nil
	}
	block, ok := s.blockByName("Pokedex")
	if !ok {
		return fmt.Errorf("the Pokédex block is missing")
	}
	base := block.Offset
	if base+gen5LangB2W2 >= len(s.data) {
		return fmt.Errorf("the Pokédex block is outside the save")
	}
	bit := int(species) - 1

	setGen5Flag(s.data[base+gen5Caught:], bit)
	setGen5Flag(s.data[base+gen5Seen:], bit)
	setGen5Flag(s.data[base+gen5Displayed:], bit)

	// Language flags cover the first 493 species.
	if species > 493 {
		return nil
	}
	langBase := gen5LangBW
	if s.family == "b2w2" {
		langBase = gen5LangB2W2
	}
	offset := base + langBase + bit/8
	if offset >= len(s.data) {
		return nil
	}
	for lang := 0; lang < gen5LangCount; lang++ {
		index := bit*gen5LangCount + lang
		setGen5Flag(s.data[base+langBase:], index)
	}
	return nil
}

func (s *SaveFile) blockByName(name string) (saveblocks.Block, bool) {
	for _, b := range s.table.Blocks {
		if b.Name == name {
			return b, true
		}
	}
	return saveblocks.Block{}, false
}

// setGen5Flag sets one bit in a little-endian bit array.
func setGen5Flag(data []byte, bitIndex int) {
	byteIndex := bitIndex >> 3
	if byteIndex >= len(data) {
		return
	}
	data[byteIndex] |= byte(1 << (bitIndex & 7))
}

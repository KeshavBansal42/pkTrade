package gen4

import (
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/modern"
)

// Generation IV keeps the Pokédex inside the general block. The layout is the
// one the reference implementation uses:
//
//	+0x000  magic
//	+0x004  region 0: caught (owned) flags, 0x40 bytes
//	+0x044  region 1: seen flags
//	+0x084  region 2: first seen gender
//	+0x0C4  region 3: second seen gender
//	+0x104  Spinda spot pattern
//	+0x108  form flags
//	+0x128  language flags (Sinnoh) / +0x144 (HeartGold/SoulSilver)
const (
	gen4RegionSize = 0x40
	gen4Caught     = 0x04
	gen4Seen       = gen4Caught + gen4RegionSize
	gen4Gender1    = gen4Caught + 2*gen4RegionSize
	gen4Gender2    = gen4Caught + 3*gen4RegionSize
	gen4Forms      = 0x108
	gen4LangDP     = gen4Forms + 0x20
	gen4LangHGSS   = gen4Forms + 0x3C
	gen4LangCount  = 6
)

// dpLangSpecies are the species that keep a per-language dex entry in the
// Sinnoh games; the National dex is a separate unlock there.
var dpLangSpecies = []uint16{23, 25, 54, 77, 120, 129, 202, 214, 215, 216, 228, 278, 287, 315}

// dexOffset is where the Pokédex starts inside the general block.
func (s *SaveFile) dexOffset() int {
	switch s.format.family {
	case "dp":
		return 0x12DC
	case "pt":
		return 0x1328
	case "hgss":
		return 0x12B8
	}
	return 0
}

func (s *SaveFile) dexBase() int {
	return s.generalBase + s.dexOffset()
}

// MarkDex records the received Pokémon in the Pokédex.
func (s *SaveFile) MarkDex(incoming core.Pokemon) error {
	p, ok := incoming.(*modern.Pokemon)
	if !ok {
		return fmt.Errorf("received record is not a Gen 4 Pokémon")
	}
	species := p.NationalID()
	if species == 0 || species > 493 {
		// The Sinnoh games only keep a dex for the first 493 species.
		return nil
	}
	base := s.dexBase()
	if base+gen4Forms >= len(s.data) {
		return fmt.Errorf("the Pokédex area is outside the save")
	}
	bit := int(species) - 1

	// Owned and seen.
	setFlag(s.data[base+gen4Caught:], bit, true)
	setFlag(s.data[base+gen4Seen:], bit, true)

	// The gender the Pokémon was seen as is written to both gender regions
	// when the species is seen for the first time.
	gender := byte(0) // male or genderless
	setFlag(s.data[base+gen4Gender1:], bit, gender == 1)
	setFlag(s.data[base+gen4Gender2:], bit, gender == 1)

	// Language flags: every language the dex knows about is marked, which is
	// what the reference implementation does when a species is caught.
	langBase := gen4LangDP
	if s.format.family == "hgss" {
		langBase = gen4LangHGSS
	}
	langIndex := int(species)
	if s.format.family == "dp" {
		langIndex = 0
		for i, sp := range dpLangSpecies {
			if sp == species {
				langIndex = i + 1
			}
		}
		if langIndex == 0 {
			// The Sinnoh games only keep language flags for a handful of
			// National dex species.
			return nil
		}
	}
	offset := base + langBase + langIndex
	if offset >= len(s.data) {
		return nil
	}
	for lang := 0; lang < gen4LangCount; lang++ {
		setFlag(s.data[offset:], lang, true)
	}
	return nil
}

// setFlag sets one bit in a little-endian bit array.
func setFlag(data []byte, bitIndex int, value bool) {
	byteIndex := bitIndex >> 3
	if byteIndex >= len(data) {
		return
	}
	mask := byte(1 << (bitIndex & 7))
	if value {
		data[byteIndex] |= mask
	} else {
		data[byteIndex] &^= mask
	}
}

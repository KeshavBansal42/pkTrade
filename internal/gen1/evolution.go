package gen1

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

var tradeEvolutions = map[uint16]uint16{
	64: 65, // Kadabra -> Alakazam
	67: 68, // Machoke -> Machamp
	75: 76, // Graveler -> Golem
	93: 94, // Haunter -> Gengar
}

func Evolve(p *Pokemon) (*Pokemon, *core.Evolution, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("no Pokémon record")
	}
	fromNational := p.NationalID()
	target, ok := tradeEvolutions[fromNational]
	if !ok {
		return p.clone(), nil, nil
	}
	targetInternal := pokemon.Internal1(target)
	if targetInternal == 0 {
		return nil, nil, fmt.Errorf("no Gen 1 species mapping for National Dex %d", target)
	}
	out := p.clone()
	out.SpeciesID = uint8(targetInternal)
	out.RawData[0x00] = uint8(targetInternal)
	if out.NicknameStr == pokemon.Name(fromNational) {
		out.NicknameRaw = encodeGen1String(pokemon.Name(target), NameLength)
		out.NicknameStr = pokemon.Name(target)
	}
	if len(out.RawData) == PartyRecordSize {
		// The old maximum HP lives in the party stats block; the record only
		// keeps the current HP next to the species.
		oldMax := int(binary.BigEndian.Uint16(out.RawData[0x22:]))
		oldHP := int(out.CurrentHP)
		out.RawData[0x21] = out.LevelVal
		stats, err := out.partyStats()
		if err != nil {
			return nil, nil, err
		}
		binaryPut := func(offset int, value uint16) {
			out.RawData[offset] = byte(value >> 8)
			out.RawData[offset+1] = byte(value)
		}
		binaryPut(0x22, stats[0])
		binaryPut(0x24, stats[1])
		binaryPut(0x26, stats[2])
		binaryPut(0x28, stats[3])
		binaryPut(0x2A, stats[4])
		hp := int(stats[0])
		if oldMax > 0 && oldHP > 0 {
			hp = oldHP * int(stats[0]) / oldMax
		}
		if hp > int(stats[0]) {
			hp = int(stats[0])
		}
		out.CurrentHP = uint16(hp)
		out.RawData[0x01] = byte(out.CurrentHP >> 8)
		out.RawData[0x02] = byte(out.CurrentHP)
	}
	return out, &core.Evolution{
		From: fromNational, To: target,
		FromName: pokemon.Name(fromNational), ToName: pokemon.Name(target),
	}, nil
}

func (s *SaveFile) FixupReceived(incoming core.Pokemon) (core.Pokemon, []core.Evolution, error) {
	p, ok := incoming.(*Pokemon)
	if !ok {
		return nil, nil, fmt.Errorf("received record is not a Gen 1 Pokémon")
	}
	evolved, evolution, err := Evolve(p)
	if err != nil {
		return nil, nil, err
	}
	if evolution == nil {
		return evolved, nil, nil
	}
	return evolved, []core.Evolution{*evolution}, nil
}

func (s *SaveFile) MarkDex(incoming core.Pokemon) error {
	p, ok := incoming.(*Pokemon)
	if !ok {
		return fmt.Errorf("received record is not a Gen 1 Pokémon")
	}
	national := p.NationalID()
	if national == 0 {
		return nil
	}
	bit := int(national) - 1
	if bit < 0 || bit >= 151 {
		return fmt.Errorf("National Dex %d is outside the Gen 1 Pokédex", national)
	}
	mask := byte(1 << (bit % 8))
	s.data[0x25A3+bit/8] |= mask // caught
	s.data[0x25B6+bit/8] |= mask // seen
	s.recomputeChecksum()
	return nil
}

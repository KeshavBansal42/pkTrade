package gen2

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

type TradeEvo struct {
	Target uint16
	Item   uint16
}

// Generation II uses its own item numbering, which shares nothing with the
// Generation III and later lists: a King's Rock is item 82 here, not 187.
var tradeEvolutions = map[uint16][]TradeEvo{
	64:  {{Target: 65}},             // Kadabra -> Alakazam
	67:  {{Target: 68}},             // Machoke -> Machamp
	75:  {{Target: 76}},             // Graveler -> Golem
	93:  {{Target: 94}},             // Haunter -> Gengar
	105: {{Target: 106}},            // Hitmonlee -> Hitmontop
	61:  {{Target: 186, Item: 82}},  // Poliwhirl + King's Rock -> Politoed
	79:  {{Target: 199, Item: 82}},  // Slowpoke + King's Rock -> Slowking
	95:  {{Target: 208, Item: 143}}, // Onix + Metal Coat -> Steelix
	123: {{Target: 212, Item: 143}}, // Scyther + Metal Coat -> Scizor
	117: {{Target: 230, Item: 151}}, // Seadra + Dragon Scale -> Kingdra
	137: {{Target: 233, Item: 172}}, // Porygon + Up-Grade -> Porygon2
}

var itemNames = map[uint16]string{
	82:  "King's Rock",
	143: "Metal Coat",
	151: "Dragon Scale",
	172: "Up-Grade",
}

func Evolve(p *Pokemon) (*Pokemon, *core.Evolution, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("no Pokémon record")
	}
	fromNational := p.NationalID()
	rules, ok := tradeEvolutions[fromNational]
	if !ok {
		return p.clone(), nil, nil
	}
	var chosen *TradeEvo
	for i := range rules {
		if rules[i].Item == 0 || rules[i].Item == uint16(p.HeldItem) {
			chosen = &rules[i]
			break
		}
	}
	if chosen == nil {
		return p.clone(), nil, nil
	}
	if chosen.Target == 0 || chosen.Target > 251 {
		return nil, nil, fmt.Errorf("no Gen 2 species mapping for National Dex %d", chosen.Target)
	}
	out := p.clone()
	out.SpeciesID = uint8(chosen.Target)
	out.RawData[0x00] = uint8(chosen.Target)
	if chosen.Item != 0 {
		out.HeldItem = 0
		out.RawData[0x01] = 0
	}
	if out.NicknameStr == pokemon.Name(fromNational) {
		out.NicknameRaw = encodeGen2String(pokemon.Name(chosen.Target), NameLength)
		out.NicknameStr = pokemon.Name(chosen.Target)
	}
	if len(out.RawData) == PartyRecordSize {
		oldMax := int(binary.BigEndian.Uint16(out.RawData[0x24:]))
		oldHP := int(binary.BigEndian.Uint16(out.RawData[0x22:]))
		stats, err := out.partyStats()
		if err != nil {
			return nil, nil, err
		}
		put := func(offset int, value uint16) { binary.BigEndian.PutUint16(out.RawData[offset:], value) }
		put(0x24, stats[0])
		put(0x26, stats[1])
		put(0x28, stats[2])
		put(0x2A, stats[3])
		put(0x2C, stats[4])
		put(0x2E, stats[5])
		hp := int(stats[0])
		if oldMax > 0 {
			hp = oldHP * int(stats[0]) / oldMax
		}
		if hp > int(stats[0]) {
			hp = int(stats[0])
		}
		put(0x22, uint16(hp))
	}
	return out, &core.Evolution{
		From: fromNational, To: chosen.Target, Item: chosen.Item,
		ItemName: itemNames[chosen.Item],
		FromName: pokemon.Name(fromNational), ToName: pokemon.Name(chosen.Target),
	}, nil
}

func (s *SaveFile) FixupReceived(incoming core.Pokemon) (core.Pokemon, []core.Evolution, error) {
	p, ok := incoming.(*Pokemon)
	if !ok {
		return nil, nil, fmt.Errorf("received record is not a Gen 2 Pokémon")
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
		return fmt.Errorf("received record is not a Gen 2 Pokémon")
	}
	national := p.NationalID()
	if national == 0 {
		return nil
	}
	bit := int(national) - 1
	if bit < 0 || bit >= 251 {
		return fmt.Errorf("National Dex %d is outside the Gen 2 Pokédex", national)
	}
	mask := byte(1 << (bit % 8))
	caught, seen := 0x2A4C, 0x2A6C
	if s.format.label == crystalFormat.label {
		caught, seen = 0x2A27, 0x2A47
	}
	s.data[caught+bit/8] |= mask
	s.data[seen+bit/8] |= mask
	s.recomputeChecksums()
	return nil
}

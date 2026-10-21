package gen3

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

// TradeEvo is one trade evolution rule, keyed by National Dex ID.
type TradeEvo struct {
	Target uint16
	Item   uint16
}

var tradeEvolutions = map[uint16][]TradeEvo{
	64:  {{Target: 65}},                                       // Kadabra -> Alakazam
	67:  {{Target: 68}},                                       // Machoke -> Machamp
	75:  {{Target: 76}},                                       // Graveler -> Golem
	93:  {{Target: 94}},                                       // Haunter -> Gengar
	105: {{Target: 106}},                                      // Hitmonlee -> Hitmontop
	61:  {{Target: 186, Item: 187}},                           // Poliwhirl + King's Rock -> Politoed
	79:  {{Target: 199, Item: 187}},                           // Slowpoke + King's Rock -> Slowking
	95:  {{Target: 208, Item: 199}},                           // Onix + Metal Coat -> Steelix
	123: {{Target: 212, Item: 199}},                           // Scyther + Metal Coat -> Scizor
	117: {{Target: 230, Item: 201}},                           // Seadra + Dragon Scale -> Kingdra
	137: {{Target: 233, Item: 218}},                           // Porygon + Up-Grade -> Porygon2
	366: {{Target: 367, Item: 192}, {Target: 368, Item: 193}}, // Clamperl
}

var itemNames = map[uint16]string{
	187: "King's Rock",
	192: "DeepSeaTooth",
	193: "DeepSeaScale",
	199: "Metal Coat",
	201: "Dragon Scale",
	218: "Up-Grade",
}

func EncodeGen3String(s string, length int) []byte {
	res := make([]byte, length)
	for i := range res {
		res[i] = 0xFF
	}
	for i := 0; i < len(s) && i < length; i++ {
		c := s[i]
		switch {
		case c == ' ':
			res[i] = 0x00
		case c >= '0' && c <= '9':
			res[i] = byte(0xA1 + (c - '0'))
		case c >= 'A' && c <= 'Z':
			res[i] = byte(0xBB + (c - 'A'))
		case c >= 'a' && c <= 'z':
			res[i] = byte(0xD5 + (c - 'a'))
		default:
			res[i] = 0xAC
		}
	}
	return res
}

// Evolve returns a copy of p with any applicable trade evolution applied.
// Item-based evolutions consume the held item. Stats are rebuilt for party
// records; boxed records only contain the stored fields and are left compact.
func Evolve(p *Pokemon) (*Pokemon, *core.Evolution, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("no Pokémon record")
	}
	if p.National == 0 {
		return p.clone(), nil, nil
	}
	rules, ok := tradeEvolutions[p.National]
	if !ok {
		return p.clone(), nil, nil
	}
	var chosen *TradeEvo
	for i := range rules {
		rule := &rules[i]
		if rule.Item == 0 {
			chosen = rule
			break
		}
		if rule.Item == p.HeldItem {
			chosen = rule
			break
		}
	}
	if chosen == nil {
		return p.clone(), nil, nil
	}

	out := p.clone()
	blocks, err := out.blocks()
	if err != nil {
		return nil, nil, err
	}
	targetInternal := pokemon.Internal3(chosen.Target)
	if targetInternal == 0 {
		return nil, nil, fmt.Errorf("no Gen 3 species mapping for National Dex %d", chosen.Target)
	}

	oldDefault := pokemon.Name(p.National)
	binary.LittleEndian.PutUint16(blocks.growth[0:2], targetInternal)
	if chosen.Item != 0 {
		binary.LittleEndian.PutUint16(blocks.growth[2:4], 0)
		out.HeldItem = 0
	}
	out.writeBlocks(blocks)

	if out.Nickname == oldDefault {
		copy(out.RawData[0x08:0x12], EncodeGen3String(pokemon.Name(chosen.Target), 10))
		out.Nickname = pokemon.Name(chosen.Target)
	}
	out.Species = targetInternal
	out.National = chosen.Target
	if level := pokemon.LevelFromExp(pokemon.Growth3[chosen.Target], out.Experience); level > 0 {
		out.Level = level
	}

	// Party records carry level and calculated stats after the encrypted
	// block. Rebuild them from the evolved species. Boxed records do not.
	if len(out.RawData) == 100 {
		oldMaxHP := binary.LittleEndian.Uint16(out.RawData[0x58:])
		oldHP := binary.LittleEndian.Uint16(out.RawData[0x56:])
		out.RawData[0x54] = out.Level
		stats, err := out.partyStats()
		if err != nil {
			return nil, nil, err
		}
		newHP := stats[0]
		if oldMaxHP > 0 {
			newHP = uint16(uint32(oldHP) * uint32(stats[0]) / uint32(oldMaxHP))
			if newHP > stats[0] {
				newHP = stats[0]
			}
		}
		binary.LittleEndian.PutUint16(out.RawData[0x56:], newHP)
		binary.LittleEndian.PutUint16(out.RawData[0x58:], stats[0])
		binary.LittleEndian.PutUint16(out.RawData[0x5A:], stats[1])
		binary.LittleEndian.PutUint16(out.RawData[0x5C:], stats[2])
		binary.LittleEndian.PutUint16(out.RawData[0x5E:], stats[3])
		binary.LittleEndian.PutUint16(out.RawData[0x60:], stats[4])
		binary.LittleEndian.PutUint16(out.RawData[0x62:], stats[5])
	}

	evolution := &core.Evolution{
		From:     p.National,
		To:       chosen.Target,
		Item:     chosen.Item,
		ItemName: itemNames[chosen.Item],
		FromName: pokemon.Name(p.National),
		ToName:   pokemon.Name(chosen.Target),
	}
	return out, evolution, nil
}

func (p *Pokemon) clone() *Pokemon {
	clone := *p
	clone.RawData = append([]byte(nil), p.RawData...)
	return &clone
}

func (s *SaveFile) FixupReceived(incoming core.Pokemon) (core.Pokemon, []core.Evolution, error) {
	p, ok := incoming.(*Pokemon)
	if !ok {
		return nil, nil, fmt.Errorf("received record is not a Gen 3 Pokémon")
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

// MarkDex sets the owned and seen bits for the received Pokémon in FRLG's
// three seen-bitmaps and the owned-bitmap. The National Dex unlock fields are
// deliberately left alone.
func (s *SaveFile) MarkDex(incoming core.Pokemon) error {
	p, ok := incoming.(*Pokemon)
	if !ok {
		return fmt.Errorf("received record is not a Gen 3 Pokémon")
	}
	if p.National == 0 {
		return nil
	}
	if s.ActiveSlot == nil {
		return fmt.Errorf("active save slot is missing")
	}
	bit := int(p.National) - 1
	if bit < 0 || bit >= 49*8 {
		return fmt.Errorf("National Dex %d is outside the FRLG bitmap", p.National)
	}
	byteIndex := bit / 8
	mask := byte(1 << (bit % 8))

	owned, ok := s.ActiveSlot.Sections[0]
	if !ok || len(owned) < 0x28+49 {
		return fmt.Errorf("owned Pokédex bitmap is missing")
	}
	owned[0x28+byteIndex] |= mask

	seenA, ok := s.ActiveSlot.Sections[0]
	if !ok || len(seenA) < 0x5C+49 {
		return fmt.Errorf("seen Pokédex bitmap A is missing")
	}
	seenA[0x5C+byteIndex] |= mask

	seenB, ok := s.ActiveSlot.Sections[1]
	if !ok || len(seenB) < 0x5F8+49 {
		return fmt.Errorf("seen Pokédex bitmap B is missing")
	}
	seenB[0x5F8+byteIndex] |= mask

	seenC, ok := s.ActiveSlot.Sections[4]
	if !ok || len(seenC) < 0xB98+49 {
		return fmt.Errorf("seen Pokédex bitmap C is missing")
	}
	seenC[0xB98+byteIndex] |= mask
	return nil
}

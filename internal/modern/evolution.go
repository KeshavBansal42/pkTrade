package modern

import (
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

// Item IDs are shared by Gen 4 onward for the evolution items below.
const (
	ItemKingsRock    = 221
	ItemUpGrade      = 252
	ItemDeepSeaTooth = 226
	ItemDeepSeaScale = 227
	ItemMetalCoat    = 233
	ItemDragonScale  = 235
	ItemProtector    = 321
	ItemElectirizer  = 322
	ItemMagmarizer   = 323
	ItemDubiousDisc  = 324
	ItemReaperCloth  = 325
	ItemRazorClaw    = 326
	ItemRazorFang    = 327
	ItemPrismScale   = 537
	ItemWhippedDream = 646
	ItemSachet       = 647
)

var itemNames = map[uint16]string{
	ItemKingsRock: "King's Rock", ItemUpGrade: "Up-Grade",
	ItemDeepSeaTooth: "Deep Sea Tooth", ItemDeepSeaScale: "Deep Sea Scale",
	ItemMetalCoat: "Metal Coat", ItemDragonScale: "Dragon Scale",
	ItemProtector: "Protector", ItemElectirizer: "Electirizer",
	ItemMagmarizer: "Magmarizer", ItemDubiousDisc: "Dubious Disc",
	ItemReaperCloth: "Reaper Cloth", ItemRazorClaw: "Razor Claw",
	ItemRazorFang: "Razor Fang", ItemPrismScale: "Prism Scale",
	ItemWhippedDream: "Whipped Dream", ItemSachet: "Sachet",
}

type rule struct {
	to   uint16
	item uint16
	min  int
}

var evolutions = map[uint16][]rule{
	64:  {{to: 65}},
	105: {{to: 106}}, // Hitmonlee -> Hitmontop
	299: {{to: 300}}, // Nosepass -> Probopass
	67:  {{to: 68}},
	75:  {{to: 76}},
	93:  {{to: 94}},
	61:  {{to: 186, item: ItemKingsRock}},
	79:  {{to: 199, item: ItemKingsRock}},
	95:  {{to: 208, item: ItemMetalCoat}},
	123: {{to: 212, item: ItemMetalCoat}},
	117: {{to: 230, item: ItemDragonScale}},
	137: {{to: 233, item: ItemUpGrade}},
	233: {{to: 474, item: ItemDubiousDisc}},
	356: {{to: 477, item: ItemReaperCloth}},
	112: {{to: 464, item: ItemProtector}},
	125: {{to: 466, item: ItemElectirizer}},
	126: {{to: 467, item: ItemMagmarizer}},
	366: {{to: 367, item: ItemDeepSeaTooth}, {to: 368, item: ItemDeepSeaScale}},
	349: {{to: 350, item: ItemPrismScale, min: 5}},
	525: {{to: 526, min: 5}},
	533: {{to: 534, min: 5}},
	708: {{to: 709, min: 6}},
	710: {{to: 711, min: 6}},
	682: {{to: 683, item: ItemSachet, min: 6}},
	684: {{to: 685, item: ItemWhippedDream, min: 6}},
}

// Evolve applies a trade evolution to p using the destination generation's
// rules. The input is not modified.
func Evolve(p *Pokemon) (*Pokemon, *core.Evolution, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("no Pokémon record")
	}
	from := p.NationalID()
	rules, ok := evolutions[from]
	if !ok {
		return p.Clone(), nil, nil
	}
	gen := p.Format.Generation
	var chosen *rule
	for i := range rules {
		r := &rules[i]
		if gen < r.min {
			continue
		}
		if r.item == 0 || r.item == p.HeldItem {
			chosen = r
			break
		}
	}
	if chosen == nil {
		return p.Clone(), nil, nil
	}
	out := p.Clone()
	if err := out.SetSpecies(chosen.to); err != nil {
		return nil, nil, err
	}
	out.SetDefaultNicknameIfNeeded(from, chosen.to)
	if chosen.item != 0 {
		out.SetItem(0)
	}
	return out, &core.Evolution{
		From: from, To: chosen.to, Item: chosen.item, ItemName: itemNames[chosen.item],
		FromName: pokemon.Name(from), ToName: pokemon.Name(chosen.to),
	}, nil
}

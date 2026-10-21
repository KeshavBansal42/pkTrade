package modern

import (
	"encoding/binary"
	"fmt"
	"testing"
)

// itemEvolutionCase describes one item-driven trade evolution.
type itemEvolutionCase struct {
	name     string
	from     uint16
	item     uint16
	to       uint16
	itemName string
}

func itemEvolutions() []itemEvolutionCase {
	return []itemEvolutionCase{
		{"Poliwhirl", 61, ItemKingsRock, 186, "King's Rock"},
		{"Slowpoke", 79, ItemKingsRock, 199, "King's Rock"},
		{"Onix", 95, ItemMetalCoat, 208, "Metal Coat"},
		{"Scyther", 123, ItemMetalCoat, 212, "Metal Coat"},
		{"Seadra", 117, ItemDragonScale, 230, "Dragon Scale"},
		{"Porygon", 137, ItemUpGrade, 233, "Up-Grade"},
		{"Porygon2", 233, ItemDubiousDisc, 474, "Dubious Disc"},
		{"Dusclops", 356, ItemReaperCloth, 477, "Reaper Cloth"},
		{"Rhydon", 112, ItemProtector, 464, "Protector"},
		{"Electabuzz", 125, ItemElectirizer, 466, "Electirizer"},
		{"Magmar", 126, ItemMagmarizer, 467, "Magmarizer"},
		{"Clamperl", 366, ItemDeepSeaTooth, 367, "Deep Sea Tooth"},
		{"Clamperl", 366, ItemDeepSeaScale, 368, "Deep Sea Scale"},
	}
}

func recordWithSpecies(t *testing.T, format Format, species uint16, item uint16) *Pokemon {
	t.Helper()
	raw := make([]byte, format.PartySize)
	binary.LittleEndian.PutUint32(raw[format.PID:], 0x12345678)
	binary.LittleEndian.PutUint16(raw[format.Species:], species)
	binary.LittleEndian.PutUint16(raw[format.Item:], item)
	raw[format.IV+3] = 0 // no egg flag
	encrypted := EncryptRecord(format, raw)
	p, err := Parse(format, encrypted)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return p
}

func TestItemEvolutionsUseTheRightItem(t *testing.T) {
	for _, tc := range itemEvolutions() {
		t.Run(tc.name, func(t *testing.T) {
			p := recordWithSpecies(t, Format6, tc.from, tc.item)
			evolved, evo, err := Evolve(p)
			if err != nil {
				t.Fatalf("evolve: %v", err)
			}
			if evo == nil {
				t.Fatalf("%d with item %d did not evolve", tc.from, tc.item)
			}
			if evo.To != tc.to {
				t.Errorf("evolved to %d, want %d", evo.To, tc.to)
			}
			if evo.ItemName != tc.itemName {
				t.Errorf("item name = %q, want %q", evo.ItemName, tc.itemName)
			}
			if evolved.GetHeldItem() != 0 {
				t.Errorf("the held item was not consumed: %d", evolved.GetHeldItem())
			}
			if evolved.GetSpecies() != tc.to {
				t.Errorf("species = %d, want %d", evolved.GetSpecies(), tc.to)
			}
			if p.GetSpecies() != tc.from {
				t.Errorf("the source record was mutated: %d", p.GetSpecies())
			}
		})
	}
}

func TestWrongItemDoesNotEvolve(t *testing.T) {
	for _, tc := range itemEvolutions() {
		t.Run(tc.name, func(t *testing.T) {
			p := recordWithSpecies(t, Format6, tc.from, 0)
			_, evo, err := Evolve(p)
			if err != nil {
				t.Fatalf("evolve: %v", err)
			}
			if evo != nil {
				t.Errorf("%d evolved without the item: %+v", tc.from, evo)
			}
		})
	}
}

func TestItemlessTradeEvolutions(t *testing.T) {
	cases := []struct {
		from uint16
		to   uint16
	}{
		{64, 65},   // Kadabra -> Alakazam
		{67, 68},   // Machoke -> Machamp
		{75, 76},   // Graveler -> Golem
		{93, 94},   // Haunter -> Gengar
		{525, 526}, // Boldore -> Gigalith (Gen 5 and later)
		{533, 534}, // Gurdurr -> Conkeldurr (Gen 5 and later)
		{105, 106}, // Hitmonlee -> Hitmontop
		{299, 300}, // Nosepass -> Probopass
		{708, 709}, // Phantump -> Trevenant (Gen 6 and later)
		{710, 711}, // Pumpkaboo -> Gourgeist (Gen 6 and later)
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d", tc.from), func(t *testing.T) {
			p := recordWithSpecies(t, Format6, tc.from, 0)
			_, evo, err := Evolve(p)
			if err != nil {
				t.Fatalf("evolve: %v", err)
			}
			if evo == nil {
				t.Fatalf("%d did not evolve", tc.from)
			}
			if evo.To != tc.to {
				t.Errorf("evolved to %d, want %d", evo.To, tc.to)
			}
		})
	}
}

// The item numbering of the later generations is not the same as the
// Generation II one, so a mixed-up table would silently stop every
// item evolution from happening.
func TestGenerationItemNumbers(t *testing.T) {
	if ItemKingsRock != 221 {
		t.Errorf("King's Rock = %d, want 221", ItemKingsRock)
	}
	if ItemUpGrade != 252 {
		t.Errorf("Up-Grade = %d, want 252", ItemUpGrade)
	}
	if ItemDeepSeaTooth != 226 {
		t.Errorf("Deep Sea Tooth = %d, want 226", ItemDeepSeaTooth)
	}
	if ItemDeepSeaScale != 227 {
		t.Errorf("Deep Sea Scale = %d, want 227", ItemDeepSeaScale)
	}
	if ItemMetalCoat != 233 {
		t.Errorf("Metal Coat = %d, want 233", ItemMetalCoat)
	}
	if ItemDragonScale != 235 {
		t.Errorf("Dragon Scale = %d, want 235", ItemDragonScale)
	}
	if ItemProtector != 321 {
		t.Errorf("Protector = %d, want 321", ItemProtector)
	}
	if ItemPrismScale != 537 {
		t.Errorf("Prism Scale = %d, want 537", ItemPrismScale)
	}
	if ItemWhippedDream != 646 {
		t.Errorf("Whipped Dream = %d, want 646", ItemWhippedDream)
	}
	if ItemSachet != 647 {
		t.Errorf("Sachet = %d, want 647", ItemSachet)
	}
}

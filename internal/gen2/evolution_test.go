package gen2

import (
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

// Generation II numbers its items differently from Generation III and later:
// a King's Rock is item 82 here and 187 in Generation III, so the tables
// cannot be shared between the two.
func TestGen2ItemEvolutionUsesTheGen2ItemNumber(t *testing.T) {
	cases := []struct {
		name     string
		from     uint16
		item     uint16
		to       uint16
		itemName string
	}{
		{"Poliwhirl", 61, 82, 186, "King's Rock"},
		{"Slowpoke", 79, 82, 199, "King's Rock"},
		{"Onix", 95, 143, 208, "Metal Coat"},
		{"Scyther", 123, 143, 212, "Metal Coat"},
		{"Seadra", 117, 151, 230, "Dragon Scale"},
		{"Porygon", 137, 172, 233, "Up-Grade"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parseGen2Record(t, tc.from, byte(tc.item))
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
			if p.GetSpecies() != tc.from {
				t.Errorf("the source record was mutated: %d", p.GetSpecies())
			}
		})
	}
}

func TestGen2ItemEvolutionIgnoresTheWrongItem(t *testing.T) {
	// 187 is the Generation III King's Rock; it must not trigger in Gen 2.
	p := parseGen2Record(t, 61, 187)
	_, evo, err := Evolve(p)
	if err != nil {
		t.Fatalf("evolve: %v", err)
	}
	if evo != nil {
		t.Errorf("a Gen 3 item number triggered a Gen 2 evolution: %+v", evo)
	}
}

func TestGen2ItemlessTradeEvolutions(t *testing.T) {
	cases := []struct {
		from uint16
		to   uint16
	}{
		{64, 65},   // Kadabra -> Alakazam
		{67, 68},   // Machoke -> Machamp
		{75, 76},   // Graveler -> Golem
		{93, 94},   // Haunter -> Gengar
		{105, 106}, // Hitmonlee -> Hitmontop
	}
	for _, tc := range cases {
		p := parseGen2Record(t, tc.from, 0)
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
	}
}

func TestGen2ItemNumbers(t *testing.T) {
	expected := map[string]uint16{
		"King's Rock":  82,
		"Metal Coat":   143,
		"Dragon Scale": 151,
		"Up-Grade":     172,
	}
	for name, want := range expected {
		var got uint16
		for id, itemName := range itemNames {
			if itemName == name {
				got = id
			}
		}
		if got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}
}

func parseGen2Record(t *testing.T, species uint16, item byte) *Pokemon {
	t.Helper()
	record := gen2Record(byte(species), item, 20)
	p, err := ParsePokemon(record, gen2Name("BREE"), gen2Name("POLI"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.NationalID() != species {
		t.Fatalf("national id = %d, want %d", p.NationalID(), species)
	}
	return p
}

var _ = core.Pokemon(&Pokemon{})
var _ = pokemon.Name

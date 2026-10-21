package gen3

import (
	"encoding/binary"
	"testing"

	"pktrade/internal/pokemon"
)

// Generation III numbers its items on its own list: a King's Rock is item 187
// here, 82 in Generation II and 221 from Generation IV on.
func TestGen3ItemEvolutionUsesTheGen3ItemNumber(t *testing.T) {
	cases := []struct {
		name     string
		from     uint16
		item     uint16
		to       uint16
		itemName string
	}{
		{"Poliwhirl", 61, 187, 186, "King's Rock"},
		{"Slowpoke", 79, 187, 199, "King's Rock"},
		{"Onix", 95, 199, 208, "Metal Coat"},
		{"Scyther", 123, 199, 212, "Metal Coat"},
		{"Seadra", 117, 201, 230, "Dragon Scale"},
		{"Porygon", 137, 218, 233, "Up-Grade"},
		{"Clamperl tooth", 366, 192, 367, "DeepSeaTooth"},
		{"Clamperl scale", 366, 193, 368, "DeepSeaScale"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parseGen3Record(t, tc.from, tc.item)
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
			if p.National != tc.from {
				t.Errorf("the source record was mutated: %d", p.National)
			}
		})
	}
}

func TestGen3ItemEvolutionIgnoresTheWrongItem(t *testing.T) {
	// 221 is the Generation IV King's Rock; it must not trigger in Gen 3.
	p := parseGen3Record(t, 61, 221)
	_, evo, err := Evolve(p)
	if err != nil {
		t.Fatalf("evolve: %v", err)
	}
	if evo != nil {
		t.Errorf("a Gen 4 item number triggered a Gen 3 evolution: %+v", evo)
	}
}

func TestGen3ItemlessTradeEvolutions(t *testing.T) {
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
		p := parseGen3Record(t, tc.from, 0)
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

// parseGen3Record builds a party-sized Gen 3 record for the given species and
// held item.
func parseGen3Record(t *testing.T, national uint16, item uint16) *Pokemon {
	t.Helper()
	internal := pokemon.Internal3(national)
	if internal == 0 {
		t.Fatalf("no Gen 3 species mapping for National Dex %d", national)
	}
	raw := make([]byte, 100)
	binary.LittleEndian.PutUint32(raw[0x00:], 0x12345678) // personality value
	binary.LittleEndian.PutUint32(raw[0x04:], 0x87654321) // original trainer id
	raw[0x08] = 0xFF                                      // nickname terminator
	raw[0x0B] = 0xFF                                      // original trainer name terminator
	raw[0x54] = 20                                        // party level

	record := &Pokemon{RawData: raw, PersonalityValue: 0x12345678, OTID: 0x87654321}
	blocks, err := record.blocks()
	if err != nil {
		t.Fatalf("blocks: %v", err)
	}
	binary.LittleEndian.PutUint16(blocks.growth[0:2], internal)
	binary.LittleEndian.PutUint16(blocks.growth[2:4], item)
	binary.LittleEndian.PutUint32(blocks.misc[0:4], 0x3FFFFFFF)
	record.writeBlocks(blocks)

	p, err := ParsePokemon(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.National != national {
		t.Fatalf("national id = %d, want %d", p.National, national)
	}
	if p.GetHeldItem() != item {
		t.Fatalf("held item = %d, want %d", p.GetHeldItem(), item)
	}
	return p
}

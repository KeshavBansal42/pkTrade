package modern

import (
	"encoding/binary"
	"os"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestParseEntityFixtures(t *testing.T) {
	cases := []struct {
		file    string
		format  Format
		species uint16
		item    uint16
		level   byte
		nick    string
		ot      string
	}{
		{"pk4.bin", Format4, 25, 193, 50, "PIKACHU", "ASH"},
		{"pk5.bin", Format5, 392, 0, 75, "INFERNAPE", "BREE"},
		{"pk6.bin", Format6, 150, 213, 70, "MEWTWO", "ASH"},
		{"pk7.bin", Format7, 700, 0, 70, "XERNEAS", "ASH"},
		{"pk8.bin", Format8, 888, 0, 80, "ZACIAN", "ASH"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			raw := readFixture(t, tc.file)
			p, err := Parse(tc.format, raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if p.GetSpecies() != tc.species {
				t.Errorf("species = %d, want %d", p.GetSpecies(), tc.species)
			}
			if p.GetHeldItem() != tc.item {
				t.Errorf("item = %d, want %d", p.GetHeldItem(), tc.item)
			}
			if p.GetLevel() != tc.level {
				t.Errorf("level = %d, want %d", p.GetLevel(), tc.level)
			}
			if p.GetNickname() != tc.nick {
				t.Errorf("nickname = %q, want %q", p.GetNickname(), tc.nick)
			}
			if p.GetTrainerName() != tc.ot {
				t.Errorf("trainer = %q, want %q", p.GetTrainerName(), tc.ot)
			}
			if !p.IsChecksumValid() {
				t.Error("checksum reported invalid")
			}
			if reason := p.TradeBlockReason(); reason != "" {
				t.Errorf("unexpected trade block reason: %s", reason)
			}

			// Re-encoding the parsed record must reproduce the fixture byte for byte.
			out, err := p.Encode(len(raw))
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if len(out) != len(raw) {
				t.Fatalf("encoded length = %d, want %d", len(out), len(raw))
			}
			for i := range raw {
				if out[i] != raw[i] {
					t.Fatalf("encoded record differs at 0x%X", i)
				}
			}
		})
	}
}

func TestParseRejectsCorruptChecksum(t *testing.T) {
	raw := readFixture(t, "pk6.bin")
	corrupt := append([]byte(nil), raw...)
	corrupt[6] ^= 0xFF
	if _, err := Parse(Format6, corrupt); err == nil {
		t.Fatal("expected a checksum error")
	}
}

func TestStoredToPartyConversionKeepsSpecies(t *testing.T) {
	raw := readFixture(t, "pk6.bin")
	p, err := Parse(Format6, raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	stored, err := p.Encode(Format6.StoredSize)
	if err != nil {
		t.Fatalf("encode stored: %v", err)
	}
	back, err := Parse(Format6, stored)
	if err != nil {
		t.Fatalf("parse stored: %v", err)
	}
	if back.GetSpecies() != p.GetSpecies() {
		t.Errorf("species changed: %d -> %d", p.GetSpecies(), back.GetSpecies())
	}
	if back.GetLevel() != p.GetLevel() {
		t.Errorf("level changed: %d -> %d", p.GetLevel(), back.GetLevel())
	}
	party, err := back.Encode(Format6.PartySize)
	if err != nil {
		t.Fatalf("encode party: %v", err)
	}
	reparsed, err := Parse(Format6, party)
	if err != nil {
		t.Fatalf("parse party: %v", err)
	}
	if reparsed.GetSpecies() != p.GetSpecies() {
		t.Errorf("species changed after party conversion")
	}
	if reparsed.GetLevel() != p.GetLevel() {
		t.Errorf("level changed after party conversion: %d", reparsed.GetLevel())
	}
}

func TestEggIsRejected(t *testing.T) {
	raw := readFixture(t, "pk6.bin")
	p, err := Parse(Format6, raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	iv := binary.LittleEndian.Uint32(p.Data[Format6.IV:])
	iv |= 1 << 30
	binary.LittleEndian.PutUint32(p.Data[Format6.IV:], iv)
	p.IsEgg = true
	if reason := p.TradeBlockReason(); reason == "" {
		t.Fatal("expected eggs to be blocked")
	}
}

func TestEvolveAppliesItemEvolution(t *testing.T) {
	raw := readFixture(t, "pk6.bin")
	p, err := Parse(Format6, raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Slowpoke is 79; with a King's Rock it evolves into Slowking.
	p.SpeciesID = 79
	binary.LittleEndian.PutUint16(p.Data[Format6.Species:], 79)
	p.SetItem(ItemKingsRock)
	evolved, evo, err := Evolve(p)
	if err != nil {
		t.Fatalf("evolve: %v", err)
	}
	if evo == nil {
		t.Fatal("expected an evolution")
	}
	if evo.To != 199 {
		t.Errorf("evolved to %d, want 199", evo.To)
	}
	if evolved.GetHeldItem() != 0 {
		t.Errorf("evolution item was not consumed: %d", evolved.GetHeldItem())
	}
	if p.GetSpecies() != 79 {
		t.Errorf("input record was mutated: species %d", p.GetSpecies())
	}
}

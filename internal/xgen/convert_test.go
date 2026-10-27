package xgen

import (
	"encoding/binary"
	"os"
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/gen2"
	"pktrade/internal/modern"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../modern/testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// TestModernStability checks that exporting a record into the neutral form and
// importing it back into the same format reproduces the fields we carry.
func TestModernStability(t *testing.T) {
	cases := []struct {
		file   string
		format modern.Format
	}{
		{"pk4.bin", modern.Format4},
		{"pk5.bin", modern.Format5},
		{"pk6.bin", modern.Format6},
		{"pk7.bin", modern.Format7},
		{"pk8.bin", modern.Format8},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			raw := readFixture(t, tc.file)
			p, err := modern.Parse(tc.format, raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			first, _, err := ExportModern(p)
			if err != nil {
				t.Fatalf("export: %v", err)
			}
			back, _, err := ImportModern(first, tc.format, Box)
			if err != nil {
				t.Fatalf("import: %v", err)
			}
			second, _, err := ExportModern(back)
			if err != nil {
				t.Fatalf("re-export: %v", err)
			}
			compareNeutral(t, first, second)
		})
	}
}

func compareNeutral(t *testing.T, a, b *Neutral) {
	t.Helper()
	check := func(name string, x, y any) {
		if x != y {
			t.Errorf("%s = %v, want %v", name, x, y)
		}
	}
	check("species", a.Species, b.Species)
	check("exp", a.Exp, b.Exp)
	check("level", a.Level, b.Level)
	check("nature", a.Nature, b.Nature)
	check("pid", a.PID, b.PID)
	check("ivs", a.IVs, b.IVs)
	check("evs", a.EVs, b.EVs)
	check("moves", a.Moves, b.Moves)
	check("pp", a.MovePP, b.MovePP)
	check("ppup", a.MovePPUp, b.MovePPUp)
	check("item", a.HeldItem, b.HeldItem)
	check("ball", a.Ball, b.Ball)
	check("hasBall", a.HasBall, b.HasBall)
	check("tid", a.TID, b.TID)
	check("sid", a.SID, b.SID)
	check("nick", a.Nick, b.Nick)
	check("ot", a.OT, b.OT)
	check("gender", a.Gender, b.Gender)
	check("shiny", a.Shiny, b.Shiny)
}

func TestPIDSearchSatisfiesNatureGenderAndShiny(t *testing.T) {
	cases := []struct {
		national uint16
		nature   byte
		gender   byte
		shiny    bool
	}{
		{25, 6, 0, true},
		{25, 6, 1, false},
		{1, 0, 0, false},
		{133, 12, 1, true},
		{81, 3, 2, false},
	}
	const tid, sid = 12345, 54321
	for _, tc := range cases {
		pid, ok := buildPID3(tc.national, tid, sid, tc.nature, tc.gender, tc.shiny, true)
		if !ok {
			t.Errorf("no PID for %+v", tc)
			continue
		}
		if pid%25 != uint32(tc.nature) {
			t.Errorf("nature: pid%%25 = %d, want %d", pid%25, tc.nature)
		}
		if got := genderFromPID(tc.national, pid); got != tc.gender {
			t.Errorf("gender = %d, want %d", got, tc.gender)
		}
		if got := isShiny(tid, sid, pid); got != tc.shiny {
			t.Errorf("shiny = %v, want %v", got, tc.shiny)
		}
	}
}

func TestPIDSearchInvokedSecretIDKeepsShiny(t *testing.T) {
	pid, ok := buildPID3(25, 12345, 0, 6, 0, true, false)
	if !ok {
		t.Fatal("no PID")
	}
	if !isShiny(12345, sidForShiny(12345, pid), pid) {
		t.Error("sidForShiny did not produce a shiny personality")
	}
	if isShiny(12345, sidForNotShiny(12345, pid), pid) {
		t.Error("sidForNotShiny produced a shiny personality")
	}
}

func TestDVIVMath(t *testing.T) {
	if got := ivFromDV(dvFromIV(31)); got != 31 {
		t.Errorf("round trip of IV 31 = %d", got)
	}
	if got := ivFromDV(dvFromIV(0)); got != 1 {
		t.Errorf("round trip of IV 0 = %d, want 1", got)
	}
	if statExpFromEV(255) != 65025 {
		t.Errorf("statExpFromEV(255) = %d", statExpFromEV(255))
	}
	if statExpFromEV(300) != 65535 {
		t.Errorf("statExpFromEV should clamp, got %d", statExpFromEV(300))
	}
	if evFromStatExp(65025) != 255 {
		t.Errorf("evFromStatExp(65025) = %d", evFromStatExp(65025))
	}
}

func TestSpeciesAbsentBlocksImport(t *testing.T) {
	n := &Neutral{Species: 700, Level: 50}
	if _, _, err := ImportGen1(n, Box); err == nil {
		t.Fatal("a Gen 5 species must not be importable into Gen 1")
	}
	if _, _, err := ImportGen3(n, Box); err == nil {
		t.Fatal("a Gen 5 species must not be importable into Gen 3")
	}
	// 700 is Xerneas; Gen 6 can hold it.
	if _, _, err := ImportModern(n, modern.Format6, Box); err != nil {
		t.Fatalf("Gen 6 should accept species 700: %v", err)
	}
}

func TestEggIsRejectedByImport(t *testing.T) {
	n := &Neutral{Species: 25, Level: 5, IsEgg: true}
	if _, _, err := ImportGen3(n, Box); err != ErrEgg {
		t.Fatalf("egg import error = %v, want ErrEgg", err)
	}
}

func TestGen2RoundTrip(t *testing.T) {
	body := make([]byte, gen2.BoxRecordSize)
	body[0x00] = 25 // Pikachu
	body[0x02] = 85 // Thunderbolt
	binary.BigEndian.PutUint16(body[0x15:], 0x0A0A)
	body[0x17] = 15
	body[0x1B] = 70
	body[0x1F] = 50
	nick := gen2.EncodeString("PIKACHU", gen2.NameLength)
	ot := gen2.EncodeString("ASH", gen2.NameLength)
	p, err := gen2.ParsePokemonMarker(body, nick, ot, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	first, _, err := ExportGen2(p)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	back, _, err := ImportGen2(first, Box)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	second, _, err := ExportGen2(back)
	if err != nil {
		t.Fatalf("re-export: %v", err)
	}
	compareNeutral(t, first, second)
	if second.Friendship != 70 {
		t.Errorf("friendship = %d, want 70", second.Friendship)
	}
}

func TestConvertPreservesSpeciesAcrossGens(t *testing.T) {
	raw := readFixture(t, "pk6.bin")
	p, err := modern.Parse(modern.Format6, raw)
	if err != nil {
		t.Fatal(err)
	}
	want := p.NationalID()

	from := core.CompatibilityKey{Generation: 6, Family: "xy", Format: 6}
	to := core.CompatibilityKey{Generation: 4, Family: "dp", Format: 4}
	out, _, err := Convert(from, to, p, core.PartySlot)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if got := core.NationalSpeciesID(out); got != want {
		t.Errorf("species = %d, want %d", got, want)
	}
	if out.GetGeneration() != 4 {
		t.Errorf("generation = %d, want 4", out.GetGeneration())
	}
}

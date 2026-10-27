package xgen

import (
	"encoding/binary"
	"os"
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/gen2"
	"pktrade/internal/gen3"
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

// neutralsForNature builds two identical Pikachu records that differ only in
// nature. Nature 3 is Adamant (+Atk, -SpA); nature 15 is Modest (+SpA, -Atk).
func pikachuAtLevel50(nature byte) *Neutral {
	return &Neutral{
		Species: 25, Level: 50, Exp: 125000,
		Nature: nature, HasNature: true,
		PID: uint32(nature), HasPID: true,
		TID: 1234, SID: 5678, HasSID: true,
		Gender: genderFromPID(25, uint32(nature)),
		IVs:    [6]byte{31, 31, 31, 31, 31, 31},
		Moves:  [4]uint16{85, 0, 0, 0}, MovePP: [4]byte{15, 0, 0, 0},
		Nick: "PIKACHU", OT: "ASH",
	}
}

func TestGen3NatureAffectsTheCorrectStats(t *testing.T) {
	adamant, _, err := ImportGen3(pikachuAtLevel50(3), Party)
	if err != nil {
		t.Fatalf("adamant import: %v", err)
	}
	modest, _, err := ImportGen3(pikachuAtLevel50(15), Party)
	if err != nil {
		t.Fatalf("modest import: %v", err)
	}
	stat := func(p *gen3.Pokemon, off int) uint16 {
		return binary.LittleEndian.Uint16(p.RawData[off:])
	}
	// Gen 3 party stats: 0x5A Atk, 0x5C Def, 0x60 SpA.
	const atk, def, spa = 0x5A, 0x5C, 0x60
	if got := stat(adamant, atk); got != 82 {
		t.Errorf("Adamant Atk = %d, want 82", got)
	}
	if got := stat(adamant, spa); got != 63 {
		t.Errorf("Adamant SpA = %d, want 63", got)
	}
	if got := stat(modest, atk); got != 67 {
		t.Errorf("Modest Atk = %d, want 67", got)
	}
	if got := stat(modest, spa); got != 77 {
		t.Errorf("Modest SpA = %d, want 77", got)
	}
	// Gen 3 Pikachu has Def 30, so a level-50, 31-IV Defense is 50; neither
	// nature touches Defense.
	if stat(adamant, def) != 50 || stat(modest, def) != 50 {
		t.Errorf("Defense must be untouched by both natures, got %d and %d",
			stat(adamant, def), stat(modest, def))
	}
}

func TestGen3RoundTrip(t *testing.T) {
	n := pikachuAtLevel50(3)
	n.Moves = [4]uint16{85, 86, 0, 0}
	n.MovePP = [4]byte{15, 10, 0, 0}
	n.MovePPUp = [4]byte{1, 2, 0, 0}
	n.HeldItem = 221 // King's Rock (canonical Gen 4+ id)
	n.HasBall = true
	n.Ball = 4 // Poké Ball
	n.Friendship = 100
	n.HasFriendship = true

	g3, _, err := ImportGen3(n, Party)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	back, _, err := ExportGen3(g3)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	compareNeutral(t, n, back)
	if back.HeldItem != 221 {
		t.Errorf("held item = %d, want 221 (King's Rock)", back.HeldItem)
	}
	if back.Friendship != 100 {
		t.Errorf("friendship = %d, want 100", back.Friendship)
	}
	if !back.HasBall || back.Ball != 4 {
		t.Errorf("ball = %d,%v, want 4,true", back.Ball, back.HasBall)
	}
}

func TestGen1Stats(t *testing.T) {
	// A level-50 Pikachu with all DVs 15 and no stat experience. Attack DV 15
	// is male for Pikachu's gender ratio, so the DV is not nudged.
	n := pikachuAtLevel50(3)
	n.Gender = 0
	g1, _, err := ImportGen1(n, Party)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	stat := func(off int) uint16 { return binary.BigEndian.Uint16(g1.RawData[off:]) }
	checks := map[int]uint16{0x22: 110, 0x24: 75, 0x26: 50, 0x28: 110, 0x2A: 70}
	for off, want := range checks {
		if got := stat(off); got != want {
			t.Errorf("Gen 1 stat at 0x%X = %d, want %d", off, got, want)
		}
	}
}

func TestGen2Stats(t *testing.T) {
	n := pikachuAtLevel50(3)
	n.Gender = 0
	g2, _, err := ImportGen2(n, Party)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	stat := func(off int) uint16 { return binary.BigEndian.Uint16(g2.RawData[off:]) }
	// 0x22 MaxHP, 0x26 Atk, 0x28 Def, 0x2A Spe, 0x2C SpA, 0x2E SpD.
	checks := map[int]uint16{0x22: 110, 0x26: 75, 0x28: 50, 0x2A: 110, 0x2C: 70, 0x2E: 60}
	for off, want := range checks {
		if got := stat(off); got != want {
			t.Errorf("Gen 2 stat at 0x%X = %d, want %d", off, got, want)
		}
	}
}

func TestModernNatureAffectsTheCorrectStats(t *testing.T) {
	// Adamant Pikachu (+Atk, -SpA) at level 50 with 31 IVs and no EVs.
	n := pikachuAtLevel50(3)
	g6, _, err := ImportModern(n, modern.Format6, Party)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	f := modern.Format6
	stat := func(off int) uint16 { return binary.LittleEndian.Uint16(g6.Data[off:]) }
	if got := stat(f.Atk); got != 82 {
		t.Errorf("Atk = %d, want 82", got)
	}
	if got := stat(f.SpA); got != 63 {
		t.Errorf("SpA = %d, want 63", got)
	}
	// Gen 6 Pikachu has Def 40, so its Defense is 60 and untouched by the nature.
	if got := stat(f.Def); got != 60 {
		t.Errorf("Def = %d, want 60", got)
	}
}

func TestShinySurvivesCrossGeneration(t *testing.T) {
	p, err := modern.Parse(modern.Format6, readFixture(t, "pk6.bin"))
	if err != nil {
		t.Fatal(err)
	}
	n, _, err := ExportModern(p)
	if err != nil {
		t.Fatal(err)
	}
	n.Shiny = true
	n.SID = sidForShiny(n.TID, n.PID)

	g3, _, err := ImportGen3(n, Party)
	if err != nil {
		t.Fatalf("to Gen 3: %v", err)
	}
	ng, _, err := ExportGen3(g3)
	if err != nil {
		t.Fatal(err)
	}
	if !ng.Shiny || !isShiny(ng.TID, ng.SID, ng.PID) {
		t.Errorf("Gen 3 lost the shiny: shiny=%v pid=%08X tid=%d sid=%d", ng.Shiny, ng.PID, ng.TID, ng.SID)
	}

	g6, _, err := ImportModern(ng, modern.Format6, Party)
	if err != nil {
		t.Fatalf("to Gen 6: %v", err)
	}
	ng6, _, err := ExportModern(g6)
	if err != nil {
		t.Fatal(err)
	}
	if !ng6.Shiny {
		t.Error("Gen 6 lost the shiny")
	}

	g2, _, err := ImportGen2(ng, Box)
	if err != nil {
		t.Fatalf("to Gen 2: %v", err)
	}
	ng2, _, err := ExportGen2(g2)
	if err != nil {
		t.Fatal(err)
	}
	if !ng2.Shiny {
		t.Error("Gen 2 lost the shiny DV pattern")
	}
}

func TestEVTotalIsCapped(t *testing.T) {
	n := pikachuAtLevel50(3)
	n.EVs = [6]uint16{255, 255, 255, 255, 255, 255}
	g3, _, err := ImportGen3(n, Party)
	if err != nil {
		t.Fatalf("Gen 3 import: %v", err)
	}
	sum := 0
	for _, ev := range g3.EVs {
		sum += int(ev)
	}
	if sum > 510 {
		t.Errorf("Gen 3 EV total = %d, want <= 510", sum)
	}

	g6, _, err := ImportModern(n, modern.Format6, Party)
	if err != nil {
		t.Fatalf("Gen 6 import: %v", err)
	}
	sum = 0
	for i := 0; i < 6; i++ {
		ev := int(g6.Data[modern.Format6.EV[i]])
		if ev > 252 {
			t.Errorf("Gen 6 EV %d = %d, want <= 252", i, ev)
		}
		sum += ev
	}
	if sum > 510 {
		t.Errorf("Gen 6 EV total = %d, want <= 510", sum)
	}
}

func TestItemTranslation(t *testing.T) {
	// King's Rock is item 82 in Gen 2, 187 in Gen 3 and 221 from Gen 4 on.
	if got, ok := itemToCanonical(3, 187); !ok || got != 221 {
		t.Errorf("Gen 3 King's Rock -> canonical = %d,%v, want 221", got, ok)
	}
	if got, ok := itemForGen(3, 221); !ok || got != 187 {
		t.Errorf("canonical -> Gen 3 = %d,%v, want 187", got, ok)
	}
	if got, ok := itemForGen(2, 221); !ok || got != 82 {
		t.Errorf("canonical -> Gen 2 = %d,%v, want 82", got, ok)
	}
	if _, ok := itemForGen(1, 221); ok {
		t.Error("Gen 1 has no held items and must report the item as unmapped")
	}
	if got, ok := itemToCanonical(1, 0); !ok || got != 0 {
		t.Errorf("a zero item should always map, got %d,%v", got, ok)
	}
}

func TestGen2DoesNotBecomeShinyByAccident(t *testing.T) {
	n := pikachuAtLevel50(3)
	// IV 21 -> DV 10 for every stat, which is the shiny DV pattern; the
	// record itself is not shiny.
	n.IVs = [6]byte{21, 21, 21, 21, 21, 21}
	n.Shiny = false

	g2, _, err := ImportGen2(n, Box)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	ng2, _, err := ExportGen2(g2)
	if err != nil {
		t.Fatal(err)
	}
	if ng2.Shiny {
		t.Error("a non-shiny Pokémon turned shiny when converted into Gen 2")
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

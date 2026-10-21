// Package tradeint holds end-to-end tests for the shared trade engine. Every
// generation is imported so that the save parser registry is complete and the
// engine can reload a rewritten save while verifying it.
package tradeint

import (
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/modern"
	"pktrade/internal/testutil"

	_ "pktrade/internal/gen1"
	_ "pktrade/internal/gen2"
	_ "pktrade/internal/gen3"
	_ "pktrade/internal/gen4"
	_ "pktrade/internal/gen5"
	_ "pktrade/internal/gen6"
	_ "pktrade/internal/gen7"
	_ "pktrade/internal/gen8"
)

func loadFixture(t *testing.T, path string) core.SaveFile {
	t.Helper()
	data := testutil.ReadGzip(t, path)
	save, err := core.LoadSaveFromBytes(data)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return save
}

// setSpecies rewrites the species of a slot so that two copies of the same
// fixture become two different saves.
func setSpecies(t *testing.T, save core.SaveFile, ref core.SlotRef, species uint16) {
	t.Helper()
	store, ok := save.(core.Storage)
	if !ok {
		t.Fatalf("save does not implement storage")
	}
	current, err := store.ReadSlot(ref)
	if err != nil {
		t.Fatalf("read %s: %v", ref, err)
	}
	record, ok := current.(*modern.Pokemon)
	if !ok {
		t.Fatalf("unexpected record type %T", current)
	}
	if err := record.SetSpecies(species); err != nil {
		t.Fatalf("set species: %v", err)
	}
	if err := store.WriteSlot(ref, record); err != nil {
		t.Fatalf("write %s: %v", ref, err)
	}
}

func TestGen6PartyTrade(t *testing.T) {
	saveA := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	saveB := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	setSpecies(t, saveA, core.Party(2), 25)
	setSpecies(t, saveB, core.Party(2), 700)

	preview := core.PreviewTrade(saveA, saveB, core.Party(2), core.Party(2))
	if !preview.Supported {
		t.Fatalf("trade not supported: %s", preview.Message)
	}
	if preview.Lossy || len(preview.Losses) != 0 {
		t.Errorf("unexpected losses: %+v", preview)
	}

	result, err := core.ExecuteTrade(saveA, saveB, core.Party(2), core.Party(2))
	if err != nil {
		t.Fatalf("trade failed: %v", err)
	}

	reloadedA, err := core.LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("reload save A: %v", err)
	}
	reloadedB, err := core.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	gotA, err := reloadedA.(core.Storage).ReadSlot(core.Party(2))
	if err != nil {
		t.Fatalf("read save A: %v", err)
	}
	gotB, err := reloadedB.(core.Storage).ReadSlot(core.Party(2))
	if err != nil {
		t.Fatalf("read save B: %v", err)
	}
	if gotA.GetSpecies() != 700 {
		t.Errorf("save A party slot 2 species = %d, want 700", gotA.GetSpecies())
	}
	if gotB.GetSpecies() != 25 {
		t.Errorf("save B party slot 2 species = %d, want 25", gotB.GetSpecies())
	}
}

func TestGen6BoxTrade(t *testing.T) {
	saveA := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	saveB := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	setSpecies(t, saveA, core.Box(1, 1), 700)
	setSpecies(t, saveB, core.Party(2), 25)

	preview := core.PreviewTrade(saveA, saveB, core.Box(1, 1), core.Box(1, 1))
	if !preview.Supported {
		t.Fatalf("box trade not supported: %s", preview.Message)
	}
	result, err := core.ExecuteTrade(saveA, saveB, core.Box(1, 1), core.Box(1, 1))
	if err != nil {
		t.Fatalf("box trade failed: %v", err)
	}
	reloadedA, err := core.LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("reload save A: %v", err)
	}
	reloadedB, err := core.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	gotA, err := reloadedA.(core.Storage).ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read save A: %v", err)
	}
	gotB, err := reloadedB.(core.Storage).ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read save B: %v", err)
	}
	if gotA.GetSpecies() != 25 {
		t.Errorf("save A box species = %d, want 25", gotA.GetSpecies())
	}
	if gotB.GetSpecies() != 700 {
		t.Errorf("save B box species = %d, want 700", gotB.GetSpecies())
	}
}

func TestGen6PartyToBoxTrade(t *testing.T) {
	saveA := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	saveB := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	setSpecies(t, saveA, core.Party(2), 700)

	preview := core.PreviewTrade(saveA, saveB, core.Party(2), core.Box(1, 1))
	if !preview.Supported {
		t.Fatalf("party to box trade not supported: %s", preview.Message)
	}
	result, err := core.ExecuteTrade(saveA, saveB, core.Party(2), core.Box(1, 1))
	if err != nil {
		t.Fatalf("party to box trade failed: %v", err)
	}
	reloadedA, err := core.LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("reload save A: %v", err)
	}
	reloadedB, err := core.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	gotA, err := reloadedA.(core.Storage).ReadSlot(core.Party(2))
	if err != nil {
		t.Fatalf("read save A: %v", err)
	}
	gotB, err := reloadedB.(core.Storage).ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read save B: %v", err)
	}
	if gotA.GetSpecies() != 25 {
		t.Errorf("save A party species = %d, want 25", gotA.GetSpecies())
	}
	if gotB.GetSpecies() != 700 {
		t.Errorf("save B box species = %d, want 700", gotB.GetSpecies())
	}
	if gotB.GetLevel() != 25 {
		t.Errorf("save B box level = %d, want 25", gotB.GetLevel())
	}
}

func TestTradeEvolutionIsApplied(t *testing.T) {
	saveA := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	saveB := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	// Kadabra evolves into Alakazam when traded.
	setSpecies(t, saveA, core.Party(2), 64)
	setSpecies(t, saveB, core.Party(2), 700)

	preview := core.PreviewTrade(saveA, saveB, core.Party(2), core.Party(2))
	if !preview.Supported {
		t.Fatalf("trade not supported: %s", preview.Message)
	}
	if len(preview.Evolutions) != 1 {
		t.Fatalf("expected one evolution, got %+v", preview.Evolutions)
	}
	if preview.Evolutions[0].From != 64 || preview.Evolutions[0].To != 65 {
		t.Errorf("unexpected evolution: %+v", preview.Evolutions[0])
	}

	result, err := core.ExecuteTrade(saveA, saveB, core.Party(2), core.Party(2))
	if err != nil {
		t.Fatalf("trade failed: %v", err)
	}
	reloadedB, err := core.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	got, err := reloadedB.(core.Storage).ReadSlot(core.Party(2))
	if err != nil {
		t.Fatalf("read save B: %v", err)
	}
	if got.GetSpecies() != 65 {
		t.Errorf("save B species = %d, want 65 (Alakazam)", got.GetSpecies())
	}
}

func TestCrossGenerationTradeIsRejected(t *testing.T) {
	saveA := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	saveB := loadFixture(t, "../gen7/testdata/sm.bin.gz")
	preview := core.PreviewTrade(saveA, saveB, core.Party(1), core.Party(1))
	if preview.Supported {
		t.Fatal("cross-generation trades must not be supported")
	}
	if preview.Lossy || len(preview.Losses) != 0 {
		t.Errorf("rejected preview must not report losses: %+v", preview)
	}
}

func TestCrossFamilyTradeIsRejected(t *testing.T) {
	saveA := loadFixture(t, "../gen4/testdata/dp.bin.gz")
	saveB := loadFixture(t, "../gen4/testdata/pt.bin.gz")
	preview := core.PreviewTrade(saveA, saveB, core.Party(1), core.Party(1))
	if preview.Supported {
		t.Fatal("trading across save families must not be supported")
	}
}

func TestSameSaveIsRejected(t *testing.T) {
	saveA := loadFixture(t, "../gen8/testdata/swsh.bin.gz")
	saveB := loadFixture(t, "../gen8/testdata/swsh.bin.gz")
	preview := core.PreviewTrade(saveA, saveB, core.Party(1), core.Party(1))
	if preview.Supported {
		t.Fatal("trading a save with itself must not be supported")
	}
}

func TestEveryGenerationAdvertisesItsFamily(t *testing.T) {
	cases := []struct {
		path string
		gen  int
	}{
		{"../gen4/testdata/dp.bin.gz", 4},
		{"../gen5/testdata/bw.bin.gz", 5},
		{"../gen6/testdata/xy.bin.gz", 6},
		{"../gen7/testdata/sm.bin.gz", 7},
		{"../gen8/testdata/swsh.bin.gz", 8},
	}
	for _, tc := range cases {
		save := loadFixture(t, tc.path)
		if save.Generation() != tc.gen {
			t.Errorf("%s generation = %d, want %d", tc.path, save.Generation(), tc.gen)
		}
		key, ok := save.(core.CompatibilityProvider)
		if !ok {
			t.Fatalf("%s does not declare a compatibility key", tc.path)
		}
		if key.CompatibilityKey().Generation != tc.gen {
			t.Errorf("%s compatibility key = %+v", tc.path, key.CompatibilityKey())
		}
		caps, ok := save.(core.CapabilityProvider)
		if !ok {
			t.Fatalf("%s does not declare capabilities", tc.path)
		}
		if !caps.Capabilities().Party || !caps.Capabilities().Boxes {
			t.Errorf("%s capabilities = %+v", tc.path, caps.Capabilities())
		}
	}
}

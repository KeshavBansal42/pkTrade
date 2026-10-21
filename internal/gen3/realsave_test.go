package gen3

import (
	"os"
	"path/filepath"
	"testing"

	"pktrade/internal/core"
)

// realSavePaths are the two FireRed save files that ship with the repository.
// They are ignored by git, so the test is skipped when they are missing.
var realSavePaths = []string{
	"../../testdata/1695 - Pokemon Fire Red (U)(Independent).sav",
	"../../testdata/1695 - Pokemon Fire Red (U)(Independent)(1).sav",
}

func loadRealSaves(t *testing.T) (*SaveFile, *SaveFile) {
	t.Helper()
	for _, path := range realSavePaths {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("real save %s is not available: %v", path, err)
		}
	}
	first, err := LoadSave(realSavePaths[0])
	if err != nil {
		t.Fatalf("load %s: %v", realSavePaths[0], err)
	}
	second, err := LoadSave(realSavePaths[1])
	if err != nil {
		t.Fatalf("load %s: %v", realSavePaths[1], err)
	}
	return first, second
}

func TestRealFireRedSaveListsPartyAndBoxes(t *testing.T) {
	save, _ := loadRealSaves(t)
	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) == 0 {
		t.Fatal("expected at least one party member")
	}
	entries, err := save.ListSlots(false)
	if err != nil {
		t.Fatalf("list slots: %v", err)
	}
	if len(entries) < len(party) {
		t.Errorf("listed %d slots, want at least %d", len(entries), len(party))
	}
	for _, entry := range entries {
		if !entry.Pokemon.IsChecksumValid() {
			t.Errorf("%s has an invalid checksum", entry.Ref)
		}
	}
}

func TestRealFireRedPartyTradeRoundTrip(t *testing.T) {
	saveA, saveB := loadRealSaves(t)

	beforeA, err := saveA.GetParty()
	if err != nil {
		t.Fatalf("party of save A: %v", err)
	}
	beforeB, err := saveB.GetParty()
	if err != nil {
		t.Fatalf("party of save B: %v", err)
	}

	preview := core.PreviewTrade(saveA, saveB, core.Party(1), core.Party(1))
	if !preview.Supported {
		t.Fatalf("party trade not supported: %s", preview.Message)
	}
	result, err := core.ExecuteTrade(saveA, saveB, core.Party(1), core.Party(1))
	if err != nil {
		t.Fatalf("party trade: %v", err)
	}

	reloadedA, err := LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("reload save A: %v", err)
	}
	reloadedB, err := LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	partyA, err := reloadedA.GetParty()
	if err != nil {
		t.Fatalf("party of save A: %v", err)
	}
	partyB, err := reloadedB.GetParty()
	if err != nil {
		t.Fatalf("party of save B: %v", err)
	}
	// Each save must now hold the other save's Pokémon.
	if partyA[0].GetSpecies() != beforeB[0].GetSpecies() {
		t.Errorf("save A holds species %d, want %d", partyA[0].GetSpecies(), beforeB[0].GetSpecies())
	}
	if partyB[0].GetSpecies() != beforeA[0].GetSpecies() {
		t.Errorf("save B holds species %d, want %d", partyB[0].GetSpecies(), beforeA[0].GetSpecies())
	}
	if err := reloadedA.Validate(); err != nil {
		t.Errorf("save A is invalid after the trade: %v", err)
	}
	if err := reloadedB.Validate(); err != nil {
		t.Errorf("save B is invalid after the trade: %v", err)
	}
}

func TestRealFireRedBoxTradeRoundTrip(t *testing.T) {
	saveA, saveB := loadRealSaves(t)

	// Move the party member of save A into a box slot and trade it.
	party, err := saveA.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if err := saveA.WriteSlot(core.Box(1, 1), party[0]); err != nil {
		t.Fatalf("write box slot: %v", err)
	}
	if err := saveB.WriteSlot(core.Box(1, 1), party[0]); err != nil {
		t.Fatalf("write box slot: %v", err)
	}

	preview := core.PreviewTrade(saveA, saveB, core.Box(1, 1), core.Box(1, 1))
	if !preview.Supported {
		t.Fatalf("box trade not supported: %s", preview.Message)
	}
	result, err := core.ExecuteTrade(saveA, saveB, core.Box(1, 1), core.Box(1, 1))
	if err != nil {
		t.Fatalf("box trade: %v", err)
	}
	reloaded, err := LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("reload save A: %v", err)
	}
	boxed, err := reloaded.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read box slot: %v", err)
	}
	if boxed.GetSpecies() != party[0].GetSpecies() {
		t.Errorf("box species = %d, want %d", boxed.GetSpecies(), party[0].GetSpecies())
	}
	if err := reloaded.Validate(); err != nil {
		t.Errorf("save A is invalid after the trade: %v", err)
	}
}

func TestRealFireRedSaveIsNotRewrittenInPlace(t *testing.T) {
	saveA, _ := loadRealSaves(t)
	before, err := os.ReadFile(filepath.Clean(realSavePaths[0]))
	if err != nil {
		t.Fatalf("read save: %v", err)
	}
	if err := saveA.WriteSlot(core.Party(1), mustPartyMember(t, saveA)); err != nil {
		t.Fatalf("write party slot: %v", err)
	}
	after, err := os.ReadFile(filepath.Clean(realSavePaths[0]))
	if err != nil {
		t.Fatalf("read save: %v", err)
	}
	if string(before) != string(after) {
		t.Error("the input file was modified; trades must never write to the input")
	}
}

func mustPartyMember(t *testing.T, save *SaveFile) core.Pokemon {
	t.Helper()
	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) == 0 {
		t.Fatal("the save has no party members")
	}
	return party[0]
}

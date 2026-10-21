package tradeint

import (
	"os"
	"testing"

	"pktrade/internal/core"
)

// These tests use the two real X/Y saves that sit in the repository's
// testdata folder. They are ignored by git, so the tests skip when the files
// are not there.
const (
	realXYA = "../../testdata/main"
	realXYB = "../../testdata/main-2"
)

func loadRealXY(t *testing.T, path string) core.SaveFile {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("real save %s is not available: %v", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	save, err := core.LoadSaveFromBytes(data)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return save
}

func TestRealXYSaveIsRead(t *testing.T) {
	save := loadRealXY(t, realXYA)
	if save.Generation() != 6 {
		t.Fatalf("generation = %d, want 6", save.Generation())
	}
	info, ok := save.(core.InfoProvider)
	if !ok {
		t.Fatal("the save does not describe itself")
	}
	if info.Info().Family != "xy" {
		t.Errorf("family = %q, want xy", info.Info().Family)
	}
	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) == 0 {
		t.Fatal("the save has no party members")
	}
	store, ok := save.(core.Storage)
	if !ok {
		t.Fatal("the save has no slot storage")
	}
	if store.BoxCount() != 31 || store.BoxCapacity() != 30 {
		t.Errorf("box geometry = %dx%d, want 31x30", store.BoxCount(), store.BoxCapacity())
	}
}

func TestRealXYBoxWriteRoundTrip(t *testing.T) {
	save := loadRealXY(t, realXYA)
	store := save.(core.Storage)
	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}

	// Put a copy of the first party member into a box slot.
	if err := store.WriteSlot(core.Box(1, 1), party[0]); err != nil {
		t.Fatalf("write box slot: %v", err)
	}
	reloaded, err := core.LoadSaveFromBytes(save.WriteToBytes())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	boxed, err := reloaded.(core.Storage).ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read box slot: %v", err)
	}
	if boxed.GetSpecies() != party[0].GetSpecies() {
		t.Errorf("box species = %d, want %d", boxed.GetSpecies(), party[0].GetSpecies())
	}
	if boxed.GetLevel() != party[0].GetLevel() {
		t.Errorf("box level = %d, want %d", boxed.GetLevel(), party[0].GetLevel())
	}
	// Writing a box slot must not disturb the party.
	after, err := reloaded.GetParty()
	if err != nil {
		t.Fatalf("party after the box write: %v", err)
	}
	if len(after) != len(party) {
		t.Errorf("party size changed: %d -> %d", len(party), len(after))
	}
}

func TestRealXYBoxTrade(t *testing.T) {
	saveA := loadRealXY(t, realXYA)
	saveB := loadRealXY(t, realXYB)
	storeA := saveA.(core.Storage)
	storeB := saveB.(core.Storage)

	partyA, err := saveA.GetParty()
	if err != nil {
		t.Fatalf("party of save A: %v", err)
	}
	partyB, err := saveB.GetParty()
	if err != nil {
		t.Fatalf("party of save B: %v", err)
	}
	// Give each save a Pokémon in the first box slot.
	if err := storeA.WriteSlot(core.Box(1, 1), partyA[1]); err != nil {
		t.Fatalf("write box of save A: %v", err)
	}
	if err := storeB.WriteSlot(core.Box(1, 1), partyB[2]); err != nil {
		t.Fatalf("write box of save B: %v", err)
	}

	preview := core.PreviewTrade(saveA, saveB, core.Box(1, 1), core.Party(1))
	if !preview.Supported {
		t.Fatalf("box trade not supported: %s", preview.Message)
	}
	result, err := core.ExecuteTrade(saveA, saveB, core.Box(1, 1), core.Party(1))
	if err != nil {
		t.Fatalf("box trade: %v", err)
	}

	reloadedA, err := core.LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("reload save A: %v", err)
	}
	reloadedB, err := core.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	boxedA, err := reloadedA.(core.Storage).ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read box of save A: %v", err)
	}
	partyA2, err := reloadedA.GetParty()
	if err != nil {
		t.Fatalf("party of save A: %v", err)
	}
	partyB2, err := reloadedB.GetParty()
	if err != nil {
		t.Fatalf("party of save B: %v", err)
	}
	boxedB, err := reloadedB.(core.Storage).ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read box of save B: %v", err)
	}

	// Save A's box must hold what save B had in its party, and save B's party
	// must hold what save A had in its box. The other two slots stay put.
	if boxedA.GetSpecies() != partyB[0].GetSpecies() {
		t.Errorf("save A box species = %d, want %d", boxedA.GetSpecies(), partyB[0].GetSpecies())
	}
	if partyB2[0].GetSpecies() != partyA[1].GetSpecies() {
		t.Errorf("save B party species = %d, want %d", partyB2[0].GetSpecies(), partyA[1].GetSpecies())
	}
	if partyA2[0].GetSpecies() != partyA[0].GetSpecies() {
		t.Errorf("save A party changed: %d, want %d", partyA2[0].GetSpecies(), partyA[0].GetSpecies())
	}
	if boxedB.GetSpecies() != partyB[2].GetSpecies() {
		t.Errorf("save B box changed: %d, want %d", boxedB.GetSpecies(), partyB[2].GetSpecies())
	}
}

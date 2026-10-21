package tradeint

import (
	"os"
	"testing"

	"pktrade/internal/core"
)

// The two real Black/White saves in the repository's testdata folder. They are
// ignored by git, so the tests skip when the files are not there.
const (
	realBWA = "../../testdata/7. Reshiram Pokemon - Black Version (USA, Europe) (NDSi Enhanced).sav"
	realBWB = "../../testdata/8. Postgame Pokemon - Black Version (USA, Europe) (NDSi Enhanced).sav"

	// The Pokédex block of a Black/White save.
	realBWDexOffset = 0x21600
	// The caught flags start at this offset inside the block.
	realBWDexCaught = 0x08
)

func loadRealBW(t *testing.T, path string) core.SaveFile {
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

func TestRealBWSaveIsRead(t *testing.T) {
	save := loadRealBW(t, realBWB)
	if save.Generation() != 5 {
		t.Fatalf("generation = %d, want 5", save.Generation())
	}
	info, ok := save.(core.InfoProvider)
	if !ok {
		t.Fatal("the save does not describe itself")
	}
	if info.Info().Family != "bw" {
		t.Errorf("family = %q, want bw", info.Info().Family)
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
	if store.BoxCount() != 24 || store.BoxCapacity() != 30 {
		t.Errorf("box geometry = %dx%d, want 24x30", store.BoxCount(), store.BoxCapacity())
	}
}

func TestRealBWPartyTrade(t *testing.T) {
	saveA := loadRealBW(t, realBWA)
	saveB := loadRealBW(t, realBWB)
	beforeA, err := saveA.GetParty()
	if err != nil {
		t.Fatalf("party of save A: %v", err)
	}
	beforeB, err := saveB.GetParty()
	if err != nil {
		t.Fatalf("party of save B: %v", err)
	}

	preview := core.PreviewTrade(saveA, saveB, core.Party(2), core.Party(1))
	if !preview.Supported {
		t.Fatalf("party trade not supported: %s", preview.Message)
	}
	result, err := core.ExecuteTrade(saveA, saveB, core.Party(2), core.Party(1))
	if err != nil {
		t.Fatalf("party trade: %v", err)
	}

	reloadedA, err := core.LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("reload save A: %v", err)
	}
	reloadedB, err := core.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	afterA, err := reloadedA.GetParty()
	if err != nil {
		t.Fatalf("party of save A: %v", err)
	}
	afterB, err := reloadedB.GetParty()
	if err != nil {
		t.Fatalf("party of save B: %v", err)
	}
	if afterA[1].GetSpecies() != beforeB[0].GetSpecies() {
		t.Errorf("save A slot 2 species = %d, want %d", afterA[1].GetSpecies(), beforeB[0].GetSpecies())
	}
	if afterB[0].GetSpecies() != beforeA[1].GetSpecies() {
		t.Errorf("save B slot 1 species = %d, want %d", afterB[0].GetSpecies(), beforeA[1].GetSpecies())
	}
	// The rest of the party must not move.
	if afterA[0].GetSpecies() != beforeA[0].GetSpecies() {
		t.Errorf("save A slot 1 changed: %d", afterA[0].GetSpecies())
	}
	if afterB[1].GetSpecies() != beforeB[1].GetSpecies() {
		t.Errorf("save B slot 2 changed: %d", afterB[1].GetSpecies())
	}

	// The received species has to be in the Pokédex of the save that took it.
	bit := int(beforeB[0].GetSpecies()) - 1
	index := realBWDexOffset + realBWDexCaught + bit/8
	if result.SaveA[index]&(1<<(bit&7)) == 0 {
		t.Error("the received species was not added to the Pokédex")
	}
}

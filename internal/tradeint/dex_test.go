package tradeint

import (
	"testing"

	"pktrade/internal/core"
)

// A trade marks the received Pokémon in the Pokédex of the save that takes
// it in, so the dex has to survive the rewrite.
func TestTradeUpdatesThePokedex(t *testing.T) {
	saveA := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	saveB := loadFixture(t, "../gen6/testdata/xy.bin.gz")
	setSpecies(t, saveA, core.Party(2), 25)
	setSpecies(t, saveB, core.Party(2), 700)

	result, err := core.ExecuteTrade(saveA, saveB, core.Party(2), core.Party(2))
	if err != nil {
		t.Fatalf("trade: %v", err)
	}

	// Save B receives the Pikachu, so its dex has to know the species.
	reloaded, err := core.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatalf("reload save B: %v", err)
	}
	dex, ok := reloaded.(core.CapabilityProvider)
	if !ok || !dex.Capabilities().Dex {
		t.Fatal("the save does not report Pokédex support")
	}
	if !dexContains(t, result.SaveB, 25) {
		t.Error("the received species was not added to the Pokédex")
	}
}

// dexContains reports whether the species is marked as caught in the Pokédex
// block of a Gen 6 save.
func dexContains(t *testing.T, data []byte, species uint16) bool {
	t.Helper()
	// The Gen 6 dex block starts with the caught flags at a fixed offset
	// inside the ZukanData block; the fixture writes it at 0x15000.
	base := 0x15000 + 0x08
	bit := int(species) - 1
	index := base + bit/8
	if index >= len(data) {
		t.Fatalf("the dex offset %d is outside the save", index)
	}
	return data[index]&(1<<(bit&7)) != 0
}

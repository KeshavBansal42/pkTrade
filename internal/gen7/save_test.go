package gen7

import (
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/testutil"
)

func loadFixture(t *testing.T) *SaveFile {
	t.Helper()
	data := testutil.ReadGzip(t, "testdata/sm.bin.gz")
	save, err := parse(data)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return save
}

func TestLoadSunMoonFixture(t *testing.T) {
	save := loadFixture(t)
	if save.Generation() != 7 {
		t.Fatalf("generation = %d, want 7", save.Generation())
	}
	if info := save.Info(); info.Family != "sm" {
		t.Errorf("family = %q, want sm", info.Family)
	}
	if err := save.Validate(); err != nil {
		t.Errorf("validate: %v", err)
	}

	trainer, err := save.TrainerInfo()
	if err != nil {
		t.Fatalf("trainer: %v", err)
	}
	if trainer.Name != "ASH" {
		t.Errorf("trainer name = %q, want ASH", trainer.Name)
	}
	if trainer.TID != 0x5555 || trainer.SID != 0x6666 {
		t.Errorf("trainer ids = %04X/%04X, want 5555/6666", trainer.TID, trainer.SID)
	}

	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) != 2 {
		t.Fatalf("party size = %d, want 2", len(party))
	}
	if party[0].GetSpecies() != 700 {
		t.Errorf("party slot 1 species = %d, want 700", party[0].GetSpecies())
	}
	if party[0].GetLevel() != 70 {
		t.Errorf("party slot 1 level = %d, want 70", party[0].GetLevel())
	}
	if party[1].GetSpecies() != 774 {
		t.Errorf("party slot 2 species = %d, want 774", party[1].GetSpecies())
	}

	boxed, err := save.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("box slot: %v", err)
	}
	if boxed.GetSpecies() != 25 {
		t.Errorf("box species = %d, want 25", boxed.GetSpecies())
	}
}

func TestBoxWriteRoundTrip(t *testing.T) {
	save := loadFixture(t)
	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if err := save.WriteSlot(core.Box(30, 20), party[1]); err != nil {
		t.Fatalf("write box slot: %v", err)
	}
	reloaded, err := parse(save.WriteToBytes())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := reloaded.Validate(); err != nil {
		t.Errorf("reloaded save is invalid: %v", err)
	}
	boxed, err := reloaded.ReadSlot(core.Box(30, 20))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if boxed.GetSpecies() != 774 {
		t.Errorf("box species = %d, want 774", boxed.GetSpecies())
	}
}

func TestPartyWriteRoundTrip(t *testing.T) {
	save := loadFixture(t)
	boxed, err := save.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("box slot: %v", err)
	}
	if err := save.WriteSlot(core.Party(1), boxed); err != nil {
		t.Fatalf("write party slot: %v", err)
	}
	reloaded, err := parse(save.WriteToBytes())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	party, err := reloaded.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) != 2 {
		t.Fatalf("party size = %d, want 2", len(party))
	}
	if party[0].GetSpecies() != 25 {
		t.Errorf("party slot 1 species = %d, want 25", party[0].GetSpecies())
	}
}

func TestEmptyPartySlotIsRejected(t *testing.T) {
	save := loadFixture(t)
	if _, err := save.ReadSlot(core.Party(4)); err == nil {
		t.Fatal("expected an error for an empty party slot")
	}
}

func TestSameSaveIsRejected(t *testing.T) {
	save := loadFixture(t)
	preview := core.PreviewTrade(save, save, core.Party(1), core.Party(2))
	if preview.Supported {
		t.Fatal("trading a save with itself must not be supported")
	}
}

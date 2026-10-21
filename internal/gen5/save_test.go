package gen5

import (
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/testutil"
)

func loadFixture(t *testing.T) *SaveFile {
	t.Helper()
	data := testutil.ReadGzip(t, "testdata/bw.bin.gz")
	save, err := parse(data)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return save
}

func TestLoadBlackWhiteFixture(t *testing.T) {
	save := loadFixture(t)
	if save.Generation() != 5 {
		t.Fatalf("generation = %d, want 5", save.Generation())
	}
	if info := save.Info(); info.Family != "bw" {
		t.Errorf("family = %q, want bw", info.Family)
	}
	if err := save.Validate(); err != nil {
		t.Errorf("validate: %v", err)
	}

	trainer, err := save.TrainerInfo()
	if err != nil {
		t.Fatalf("trainer: %v", err)
	}
	if trainer.Name != "BREE" {
		t.Errorf("trainer name = %q, want BREE", trainer.Name)
	}
	if trainer.TID != 0x3333 || trainer.SID != 0x4444 {
		t.Errorf("trainer ids = %04X/%04X, want 3333/4444", trainer.TID, trainer.SID)
	}

	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) != 2 {
		t.Fatalf("party size = %d, want 2", len(party))
	}
	if party[0].GetSpecies() != 392 {
		t.Errorf("party slot 1 species = %d, want 392", party[0].GetSpecies())
	}
	if party[0].GetLevel() != 75 {
		t.Errorf("party slot 1 level = %d, want 75", party[0].GetLevel())
	}
	if party[0].GetTrainerName() != "BREE" {
		t.Errorf("party slot 1 trainer = %q, want BREE", party[0].GetTrainerName())
	}

	boxed, err := save.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("box slot: %v", err)
	}
	if boxed.GetSpecies() != 150 {
		t.Errorf("box species = %d, want 150", boxed.GetSpecies())
	}
	if boxed.GetLevel() != 40 {
		t.Errorf("box level = %d, want 40", boxed.GetLevel())
	}
}

func TestBoxWriteRoundTrip(t *testing.T) {
	save := loadFixture(t)
	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if err := save.WriteSlot(core.Box(3, 7), party[0]); err != nil {
		t.Fatalf("write box slot: %v", err)
	}
	reloaded, err := parse(save.WriteToBytes())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := reloaded.Validate(); err != nil {
		t.Errorf("reloaded save is invalid: %v", err)
	}
	boxed, err := reloaded.ReadSlot(core.Box(3, 7))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if boxed.GetSpecies() != 392 {
		t.Errorf("box species = %d, want 392", boxed.GetSpecies())
	}
	if boxed.GetLevel() != 75 {
		t.Errorf("box level = %d, want 75", boxed.GetLevel())
	}
	other, err := reloaded.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read untouched box slot: %v", err)
	}
	if other.GetSpecies() != 150 {
		t.Errorf("untouched box slot changed: species %d", other.GetSpecies())
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
	if party[0].GetSpecies() != 150 {
		t.Errorf("party slot 1 species = %d, want 150", party[0].GetSpecies())
	}
}

func TestEmptyPartySlotIsRejected(t *testing.T) {
	save := loadFixture(t)
	if _, err := save.ReadSlot(core.Party(5)); err == nil {
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

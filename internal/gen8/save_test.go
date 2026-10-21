package gen8

import (
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/testutil"
)

func loadFixture(t *testing.T) *SaveFile {
	t.Helper()
	data := testutil.ReadGzip(t, "testdata/swsh.bin.gz")
	save, err := parse(data)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return save
}

func TestLoadSwordShieldFixture(t *testing.T) {
	save := loadFixture(t)
	if save.Generation() != 8 {
		t.Fatalf("generation = %d, want 8", save.Generation())
	}
	if info := save.Info(); info.Family != "swsh" {
		t.Errorf("family = %q, want swsh", info.Family)
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
	if trainer.TID != 0x7777 || trainer.SID != 0x8888 {
		t.Errorf("trainer ids = %04X/%04X, want 7777/8888", trainer.TID, trainer.SID)
	}

	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) != 2 {
		t.Fatalf("party size = %d, want 2", len(party))
	}
	if party[0].GetSpecies() != 888 {
		t.Errorf("party slot 1 species = %d, want 888", party[0].GetSpecies())
	}
	if party[0].GetLevel() != 80 {
		t.Errorf("party slot 1 level = %d, want 80", party[0].GetLevel())
	}
	if party[1].GetSpecies() != 887 {
		t.Errorf("party slot 2 species = %d, want 887", party[1].GetSpecies())
	}

	boxed, err := save.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("box slot: %v", err)
	}
	if boxed.GetSpecies() != 25 {
		t.Errorf("box species = %d, want 25", boxed.GetSpecies())
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
	if err := save.WriteSlot(core.Box(12, 4), party[0]); err != nil {
		t.Fatalf("write box slot: %v", err)
	}
	written := save.WriteToBytes()

	reloaded, err := parse(written)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := reloaded.Validate(); err != nil {
		t.Errorf("reloaded save is invalid: %v", err)
	}
	boxed, err := reloaded.ReadSlot(core.Box(12, 4))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if boxed.GetSpecies() != 888 {
		t.Errorf("box species = %d, want 888", boxed.GetSpecies())
	}
	if boxed.GetLevel() != 80 {
		t.Errorf("box level = %d, want 80", boxed.GetLevel())
	}
	other, err := reloaded.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read untouched box slot: %v", err)
	}
	if other.GetSpecies() != 25 {
		t.Errorf("untouched box slot changed: species %d", other.GetSpecies())
	}
}

func TestPartyWriteRoundTrip(t *testing.T) {
	save := loadFixture(t)
	boxed, err := save.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("box slot: %v", err)
	}
	if err := save.WriteSlot(core.Party(2), boxed); err != nil {
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
	if party[1].GetSpecies() != 25 {
		t.Errorf("party slot 2 species = %d, want 25", party[1].GetSpecies())
	}
}

func TestWriteToBytesRoundTripsExactly(t *testing.T) {
	save := loadFixture(t)
	original := save.WriteToBytes()
	reloaded, err := parse(original)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	again := reloaded.WriteToBytes()
	if len(again) != len(original) {
		t.Fatalf("length changed: %d -> %d", len(original), len(again))
	}
	for i := range original {
		if again[i] != original[i] {
			t.Fatalf("byte %d changed on re-encode", i)
		}
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

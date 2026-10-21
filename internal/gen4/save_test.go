package gen4

import (
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/testutil"
)

func loadFixture(t *testing.T) *SaveFile {
	t.Helper()
	data := testutil.ReadGzip(t, "testdata/dp.bin.gz")
	save, err := parse(data)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return save
}

func TestLoadDiamondPearlFixture(t *testing.T) {
	save := loadFixture(t)
	if save.Generation() != 4 {
		t.Fatalf("generation = %d, want 4", save.Generation())
	}
	info := save.Info()
	if info.Family != "dp" {
		t.Errorf("family = %q, want dp", info.Family)
	}
	caps := save.Capabilities()
	if !caps.Party || !caps.Boxes || !caps.Evolution {
		t.Errorf("unexpected capabilities: %+v", caps)
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
	if trainer.TID != 0x1111 || trainer.SID != 0x2222 {
		t.Errorf("trainer ids = %04X/%04X, want 1111/2222", trainer.TID, trainer.SID)
	}

	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if len(party) != 2 {
		t.Fatalf("party size = %d, want 2", len(party))
	}
	// The fixture stores two partitions; the newer one is active.
	if party[0].GetSpecies() != 26 {
		t.Errorf("party slot 1 species = %d, want 26", party[0].GetSpecies())
	}
	if party[1].GetSpecies() != 94 {
		t.Errorf("party slot 2 species = %d, want 94", party[1].GetSpecies())
	}
	if party[0].GetLevel() != 50 {
		t.Errorf("party slot 1 level = %d, want 50", party[0].GetLevel())
	}

	boxed, err := save.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("box slot: %v", err)
	}
	if boxed.GetSpecies() != 150 {
		t.Errorf("box species = %d, want 150", boxed.GetSpecies())
	}
}

func TestBoxWriteRoundTrip(t *testing.T) {
	save := loadFixture(t)
	party, err := save.GetParty()
	if err != nil {
		t.Fatalf("party: %v", err)
	}
	if err := save.WriteSlot(core.Box(2, 5), party[0]); err != nil {
		t.Fatalf("write box slot: %v", err)
	}
	written := save.WriteToBytes()

	reloaded, err := parse(written)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	boxed, err := reloaded.ReadSlot(core.Box(2, 5))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if boxed.GetSpecies() != 26 {
		t.Errorf("box species = %d, want 26", boxed.GetSpecies())
	}
	if boxed.GetLevel() != 50 {
		t.Errorf("box level = %d, want 50", boxed.GetLevel())
	}
	other, err := reloaded.ReadSlot(core.Box(1, 1))
	if err != nil {
		t.Fatalf("read untouched box slot: %v", err)
	}
	if other.GetSpecies() != 150 {
		t.Errorf("untouched box slot changed: species %d", other.GetSpecies())
	}
	newParty, err := reloaded.GetParty()
	if err != nil {
		t.Fatalf("party after write: %v", err)
	}
	if len(newParty) != len(party) {
		t.Errorf("party size changed: %d -> %d", len(party), len(newParty))
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
	if party[1].GetSpecies() != 150 {
		t.Errorf("party slot 2 species = %d, want 150", party[1].GetSpecies())
	}
	if party[1].GetLevel() != 40 {
		t.Errorf("party slot 2 level = %d, want 40", party[1].GetLevel())
	}
}

func TestEmptyPartySlotIsRejected(t *testing.T) {
	save := loadFixture(t)
	if _, err := save.ReadSlot(core.Party(4)); err == nil {
		t.Fatal("expected an error for an empty party slot")
	}
	if err := save.WriteSlot(core.Party(4), nil); err == nil {
		t.Fatal("expected an error when writing past the end of the party")
	}
}

func TestSameSaveIsRejected(t *testing.T) {
	save := loadFixture(t)
	preview := core.PreviewTrade(save, save, core.Party(1), core.Party(2))
	if preview.Supported {
		t.Fatal("trading a save with itself must not be supported")
	}
	if preview.Lossy || len(preview.Losses) != 0 {
		t.Errorf("rejected preview must not report losses: %+v", preview)
	}
}

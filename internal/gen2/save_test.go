package gen2

import (
	"encoding/binary"
	"testing"

	"pktrade/internal/core"
)

func gen2Record(species, heldItem, level byte) []byte {
	rec := make([]byte, PartyRecordSize)
	rec[0x00] = species
	rec[0x01] = heldItem
	rec[0x1F] = level
	return rec
}

func gen2Name(s string) []byte {
	raw := make([]byte, NameLength)
	for i := range raw {
		raw[i] = 0x50
	}
	for i := 0; i < len(s) && i < NameLength; i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			raw[i] = 0x80 + (c - 'A')
		case c >= 'a' && c <= 'z':
			raw[i] = 0xA0 + (c - 'a')
		case c >= '0' && c <= '9':
			raw[i] = 0xF6 + (c - '0')
		default:
			raw[i] = c
		}
	}
	return raw
}

func gen2GSave(records [][]byte, otNames, nicks [][]byte) []byte {
	f := gsFormat
	data := make([]byte, SaveSize)

	block := make([]byte, partyBlockSize)
	block[0] = byte(len(records))
	for i, rec := range records {
		copy(block[8+i*PartyRecordSize:], rec)
		block[1+i] = rec[0]
		copy(block[296+i*NameLength:], otNames[i])
		copy(block[362+i*NameLength:], nicks[i])
	}
	block[1+len(records)] = 0xFF

	copy(data[f.partyOffset:], block)
	copy(data[f.partyOffset+f.secondaryDelta:], block)

	// The copy of the current box has to be a well-formed packed list for the
	// save to be recognised as a Generation II save.
	data[f.currentBox] = 0
	data[f.currentBox+1] = 0xFF

	binary.LittleEndian.PutUint16(data[f.primaryChecksum:], sum16(data, f.primaryStart, f.primaryEnd))
	var secondary uint16
	for _, r := range f.secondaryRanges {
		secondary += sum16(data, r[0], r[1])
	}
	binary.LittleEndian.PutUint16(data[f.secondaryChecksum:], secondary)
	return data
}

func TestGen2LoadRoundTripIsByteExact(t *testing.T) {
	recordA := gen2Record(1, 0, 10)
	recordB := gen2Record(2, 0xAD, 20)
	original := gen2GSave(
		[][]byte{recordA, recordB},
		[][]byte{gen2Name("GOLD"), gen2Name("SILVER")},
		[][]byte{gen2Name("CHIK"), gen2Name("CYNDA")},
	)

	save, err := core.LoadSaveFromBytes(append([]byte(nil), original...))
	if err != nil {
		t.Fatal(err)
	}
	if save.Generation() != 2 {
		t.Fatalf("got generation %d, want 2", save.Generation())
	}
	party, err := save.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	if len(party) != 2 {
		t.Fatalf("got %d party members, want 2", len(party))
	}
	if got := party[0].GetLevel(); got != 10 {
		t.Fatalf("level from 0x1F = %d, want 10", got)
	}
	if got := party[1].GetHeldItem(); got != 0xAD {
		t.Fatalf("held item = %#x, want 0xAD", got)
	}
	if got := party[1].GetNickname(); got != "CYNDA" {
		t.Fatalf("nickname = %q, want CYNDA", got)
	}

	if err := save.InjectParty(party); err != nil {
		t.Fatal(err)
	}
	if string(save.WriteToBytes()) != string(original) {
		t.Fatal("re-injecting the same party did not reproduce the save byte-for-byte")
	}
}

func TestGen2SwapUpdatesBothCopiesAndChecksums(t *testing.T) {
	recordA := gen2Record(1, 0, 10)
	recordB := gen2Record(2, 0, 20)
	save, err := core.LoadSaveFromBytes(gen2GSave(
		[][]byte{recordA, recordB},
		[][]byte{gen2Name("GOLD"), gen2Name("SILVER")},
		[][]byte{gen2Name("CHIK"), gen2Name("CYNDA")},
	))
	if err != nil {
		t.Fatal(err)
	}
	party, err := save.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	if err := save.InjectParty([]core.Pokemon{party[1], party[0]}); err != nil {
		t.Fatal(err)
	}

	f := gsFormat
	data := save.(*SaveFile).data
	primary := data[f.partyOffset : f.partyOffset+partyBlockSize]
	secondary := data[f.partyOffset+f.secondaryDelta : f.partyOffset+f.secondaryDelta+partyBlockSize]
	if string(primary) != string(secondary) {
		t.Fatal("secondary party copy was not kept in sync with the primary")
	}

	reloaded, err := core.LoadSaveFromBytes(save.WriteToBytes())
	if err != nil {
		t.Fatalf("written save failed checksum validation: %v", err)
	}
	got, err := reloaded.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	if string(got[0].ToBytes()) != string(recordB) || string(got[1].ToBytes()) != string(recordA) {
		t.Fatal("swapped records were not preserved exactly")
	}
}

func TestGen2InjectRejectsForeignRecordWithoutMutating(t *testing.T) {
	save, err := core.LoadSaveFromBytes(gen2GSave(
		[][]byte{gen2Record(1, 0, 10)},
		[][]byte{gen2Name("GOLD")},
		[][]byte{gen2Name("CHIK")},
	))
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), save.(*SaveFile).data...)
	if err := save.InjectParty([]core.Pokemon{&Pokemon{}}); err == nil {
		t.Fatal("expected missing preserved Gen 2 record to be rejected")
	}
	if string(save.(*SaveFile).data) != string(before) {
		t.Fatal("rejected injection mutated the save")
	}
}

package gen1

import (
	"testing"

	"pktrade/internal/core"
)

func gen1Record(species, level byte) []byte {
	rec := make([]byte, PartyRecordSize)
	rec[0x00] = species
	rec[0x21] = level
	return rec
}

func gen1Name(s string) []byte {
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

func gen1Save(records [][]byte, otNames, nicks [][]byte) []byte {
	data := make([]byte, SaveSize)
	data[partyCountOffset] = byte(len(records))
	for i, rec := range records {
		copy(data[partyRecordBase+i*PartyRecordSize:], rec)
		data[partySpeciesBase+i] = rec[0]
		copy(data[partyOTNameBase+i*NameLength:], otNames[i])
		copy(data[partyNickNameBase+i*NameLength:], nicks[i])
	}
	data[partySpeciesBase+len(records)] = 0xFF
	// The game keeps a copy of the current box list at 0x30C0; it has to be
	// a well-formed list for the save to be recognised.
	data[currentBox] = 0
	data[currentBox+1] = 0xFF
	data[checksumOffset] = mainChecksum(data)
	return data
}

func TestGen1LoadRoundTripIsByteExact(t *testing.T) {
	recordA := gen1Record(0x99, 5)
	recordB := gen1Record(0xB0, 12)
	original := gen1Save(
		[][]byte{recordA, recordB},
		[][]byte{gen1Name("RED"), gen1Name("BLUE")},
		[][]byte{gen1Name("BULBA"), gen1Name("CHAR")},
	)

	save, err := core.LoadSaveFromBytes(append([]byte(nil), original...))
	if err != nil {
		t.Fatal(err)
	}
	if save.Generation() != 1 {
		t.Fatalf("got generation %d, want 1", save.Generation())
	}
	party, err := save.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	if len(party) != 2 {
		t.Fatalf("got %d party members, want 2", len(party))
	}
	if got := party[0].GetLevel(); got != 5 {
		t.Fatalf("level from 0x21 = %d, want 5", got)
	}
	if got := party[1].GetNickname(); got != "CHAR" {
		t.Fatalf("nickname = %q, want CHAR", got)
	}

	if err := save.InjectParty(party); err != nil {
		t.Fatal(err)
	}
	if string(save.WriteToBytes()) != string(original) {
		t.Fatal("re-injecting the same party did not reproduce the save byte-for-byte")
	}
}

func TestGen1SwapPreservesRecordsAndChecksum(t *testing.T) {
	recordA := gen1Record(0x99, 5)
	recordB := gen1Record(0xB0, 12)
	save, err := core.LoadSaveFromBytes(gen1Save(
		[][]byte{recordA, recordB},
		[][]byte{gen1Name("RED"), gen1Name("BLUE")},
		[][]byte{gen1Name("BULBA"), gen1Name("CHAR")},
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
	if got[0].GetNickname() != "CHAR" || got[1].GetNickname() != "BULBA" {
		t.Fatal("nicknames did not follow their records")
	}
}

func TestGen1InjectRejectsForeignRecordWithoutMutating(t *testing.T) {
	save, err := core.LoadSaveFromBytes(gen1Save(
		[][]byte{gen1Record(0x99, 5)},
		[][]byte{gen1Name("RED")},
		[][]byte{gen1Name("BULBA")},
	))
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), save.(*SaveFile).data...)
	if err := save.InjectParty([]core.Pokemon{&Pokemon{}}); err == nil {
		t.Fatal("expected missing preserved Gen 1 record to be rejected")
	}
	if string(save.(*SaveFile).data) != string(before) {
		t.Fatal("rejected injection mutated the save")
	}
}

func TestGen1ParserRejectsBadChecksum(t *testing.T) {
	data := gen1Save(
		[][]byte{gen1Record(0x99, 5)},
		[][]byte{gen1Name("RED")},
		[][]byte{gen1Name("BULBA")},
	)
	data[checksumOffset]++
	if _, err := core.LoadSaveFromBytes(data); err == nil {
		t.Fatal("gen 1 save with a bad checksum was accepted")
	}
}

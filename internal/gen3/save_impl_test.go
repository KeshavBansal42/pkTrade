package gen3

import (
	"encoding/binary"
	"testing"

	"pktrade/internal/core"
)

func makeSaveWithParty(records ...[]byte) []byte {
	data := make([]byte, 131072)
	for slotIndex, slotOffset := range []int{SlotAOffset, SlotBOffset} {
		for physicalIndex := 0; physicalIndex < SectionsPerSlot; physicalIndex++ {
			section := make([]byte, SectionSize)
			binary.LittleEndian.PutUint16(section[0xFF4:], uint16(physicalIndex))
			binary.LittleEndian.PutUint32(section[0xFF8:], Signature)
			binary.LittleEndian.PutUint32(section[0xFFC:], uint32(1-slotIndex))
			if physicalIndex == 1 {
				binary.LittleEndian.PutUint32(section[0x34:], uint32(len(records)))
				for i, record := range records {
					copy(section[0x38+i*100:], record)
				}
			}
			binary.LittleEndian.PutUint16(section[0xFF6:], sectionChecksum(section[:sectionDataSizes[uint16(physicalIndex)]]))
			copy(data[slotOffset+physicalIndex*SectionSize:], section)
		}
	}
	return data
}

func TestInjectPartyPreservesRawRecordsAndRecomputesSectionChecksum(t *testing.T) {
	recordA := makePartyRecord(3, 4)
	recordB := makePartyRecord(19, 11)
	save, err := LoadSaveFromBytes(makeSaveWithParty(recordA, recordB))
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

	written := save.WriteToBytes()
	reloaded, err := LoadSaveFromBytes(written)
	if err != nil {
		t.Fatalf("written save failed checksum validation: %v", err)
	}
	got, err := reloaded.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	if string(got[0].ToBytes()) != string(recordB) || string(got[1].ToBytes()) != string(recordA) {
		t.Fatal("injected party records were not preserved exactly")
	}
}

func TestInjectPartyRejectsNonGen3BeforeMutating(t *testing.T) {
	record := makePartyRecord(2, 4)
	save, err := LoadSaveFromBytes(makeSaveWithParty(record))
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), save.RawData...)
	if err := save.InjectParty([]core.Pokemon{&Pokemon{}}); err == nil {
		t.Fatal("expected missing preserved record error")
	}
	if string(save.RawData) != string(before) {
		t.Fatal("rejected injection mutated the save")
	}
}

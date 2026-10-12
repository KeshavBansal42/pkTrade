package mobile

import (
	"encoding/binary"
	"strings"
	"testing"

	"pktrade/internal/core"
	"pktrade/internal/gen3"
)

var testSubStructOrder = [24]string{
	"GAEM", "GAME", "GEAM", "GEMA", "GMAE", "GMEA",
	"AGEM", "AGME", "AEGM", "AEMG", "AMGE", "AMEG",
	"EGAM", "EGMA", "EAGM", "EAMG", "EMGA", "EMAG",
	"MGAE", "MGEA", "MAGE", "MAEG", "MEGA", "MEAG",
}

func makeRecord(pid uint32, ballID byte) []byte {
	record := make([]byte, 100)
	binary.LittleEndian.PutUint32(record, pid)
	otID := uint32(0x12345678)
	binary.LittleEndian.PutUint32(record[4:], otID)
	record[0x54] = 42

	var blocks [4][12]byte
	binary.LittleEndian.PutUint16(blocks[0][:], 25)
	binary.LittleEndian.PutUint16(blocks[3][2:], uint16(ballID&0x0F)<<11)
	binary.LittleEndian.PutUint16(blocks[1][:], 33)
	decrypted := make([]byte, 48)
	for i, id := range testSubStructOrder[pid%24] {
		index := strings.IndexByte("GAEM", byte(id))
		copy(decrypted[i*12:], blocks[index][:])
	}
	var sum uint32
	for i := 0; i < len(decrypted); i += 2 {
		sum += uint32(binary.LittleEndian.Uint16(decrypted[i:]))
	}
	binary.LittleEndian.PutUint16(record[0x1C:], uint16(sum))
	key := pid ^ otID
	for i := 0; i < len(decrypted); i += 4 {
		binary.LittleEndian.PutUint32(record[0x20+i:], binary.LittleEndian.Uint32(decrypted[i:])^key)
	}
	return record
}

func testSectionChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+4 <= len(data); i += 4 {
		sum += binary.LittleEndian.Uint32(data[i:])
	}
	return uint16(sum>>16) + uint16(sum)
}

func saveWithParty(records ...[]byte) []byte {
	data := make([]byte, 131072)
	for slotIndex, slotOffset := range []int{gen3.SlotAOffset, gen3.SlotBOffset} {
		for physicalIndex := 0; physicalIndex < gen3.SectionsPerSlot; physicalIndex++ {
			section := make([]byte, gen3.SectionSize)
			binary.LittleEndian.PutUint16(section[0xFF4:], uint16(physicalIndex))
			binary.LittleEndian.PutUint32(section[0xFF8:], gen3.Signature)
			binary.LittleEndian.PutUint32(section[0xFFC:], uint32(1-slotIndex))
			if physicalIndex == 1 {
				binary.LittleEndian.PutUint32(section[0x34:], uint32(len(records)))
				for i, record := range records {
					copy(section[0x38+i*100:], record)
				}
			}
			binary.LittleEndian.PutUint16(section[0xFF6:], testSectionChecksum(section[:sectionDataSize(uint16(physicalIndex))]))
			copy(data[slotOffset+physicalIndex*gen3.SectionSize:], section)
		}
	}
	return data
}

func sectionDataSize(id uint16) int {
	if id == 0 {
		return 3884
	}
	if id == 4 {
		return 3848
	}
	if id == 13 {
		return 2000
	}
	return 3968
}

func TestParsePartyJSONExposesGen3CaptureBall(t *testing.T) {
	payload, err := ParsePartyJSON(saveWithParty(makeRecord(5, 10)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `"ballId":10`) {
		t.Fatalf("ball id missing from party JSON: %s", payload)
	}
}

func TestPreviewTradeJSONUsesLowerCamelCaseContract(t *testing.T) {
	payload := PreviewTradeJSON(saveWithParty(makeRecord(1, 4)), saveWithParty(makeRecord(2, 5)), 1, 1)
	for _, key := range []string{`"supported":true`, `"lossy":false`, `"message"`, `"losses"`} {
		if !strings.Contains(payload, key) {
			t.Fatalf("preview JSON missing %s: %s", key, payload)
		}
	}
}

func TestPreviewTradeSeparatesSupportedAndUnsupportedPairs(t *testing.T) {
	gen3SaveA := saveWithParty(makeRecord(1, 4))
	gen3SaveB := saveWithParty(makeRecord(2, 5))
	preview := PreviewTrade(gen3SaveA, gen3SaveB, 1, 1)
	if !preview.Supported || preview.Lossy {
		t.Fatalf("Gen 3 pair should be lossless and supported: %+v", preview)
	}

	legacySave := make([]byte, 32768)
	preview = PreviewTrade(gen3SaveA, legacySave, 1, 1)
	if preview.Supported || preview.Lossy {
		t.Fatalf("legacy preview must not be runnable: %+v", preview)
	}
	if len(preview.Losses) != 0 {
		t.Fatalf("ambiguous 32KiB detection must not fabricate a loss list: %+v", preview.Losses)
	}
	if preview.Message == "" {
		t.Fatal("unsupported preview should explain why it is blocked")
	}
}

func TestPerformTradeCopiesFullGen3Records(t *testing.T) {
	recordA := makeRecord(2, 4)
	recordB := makeRecord(13, 11)
	result, err := PerformTrade(saveWithParty(recordA), saveWithParty(recordB), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	fileA, err := gen3.LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatal(err)
	}
	partyA, err := fileA.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	fileB, err := gen3.LoadSaveFromBytes(result.SaveB)
	if err != nil {
		t.Fatal(err)
	}
	partyB, err := fileB.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	if string(partyA[0].ToBytes()) != string(recordB) || string(partyB[0].ToBytes()) != string(recordA) {
		t.Fatal("trade did not preserve and swap complete party records")
	}
}

func TestPerformTradeRejectsUnsupportedGenerationWithoutChangingInputs(t *testing.T) {
	gen3Save := saveWithParty(makeRecord(1, 4))
	gen1Save := make([]byte, 32768)
	beforeA := append([]byte(nil), gen3Save...)
	beforeB := append([]byte(nil), gen1Save...)
	if _, err := PerformTrade(gen3Save, gen1Save, 1, 1); err == nil {
		t.Fatal("expected unsupported generation pair to be rejected")
	}
	if string(gen3Save) != string(beforeA) || string(gen1Save) != string(beforeB) {
		t.Fatal("rejected trade changed input save bytes")
	}
}

func TestPerformTradeRejectsOutOfRangePartyIndex(t *testing.T) {
	saveA := saveWithParty(makeRecord(1, 4))
	saveB := saveWithParty(makeRecord(2, 5))
	beforeA := append([]byte(nil), saveA...)
	beforeB := append([]byte(nil), saveB...)
	if _, err := PerformTrade(saveA, saveB, 2, 1); err == nil {
		t.Fatal("expected nonexistent party index to be rejected")
	}
	if string(saveA) != string(beforeA) || string(saveB) != string(beforeB) {
		t.Fatal("rejected trade changed input save bytes")
	}
}

// gen1SaveForMobile builds a minimal but valid Gen 1 save (documented layout).
func gen1SaveForMobile(records ...[]byte) []byte {
	data := make([]byte, 32768)
	data[0x2F2C] = byte(len(records))
	for i, rec := range records {
		copy(data[0x2F34+i*44:], rec)
		data[0x2F2D+i] = rec[0]
	}
	data[0x2F2D+len(records)] = 0xFF
	var sum byte
	for i := 0x2598; i <= 0x3522; i++ {
		sum += data[i]
	}
	data[0x3523] = sum ^ 0xFF
	return data
}

func gen1RecordForMobile(species, level byte) []byte {
	rec := make([]byte, 44)
	rec[0x00] = species
	rec[0x21] = level
	return rec
}

func TestPerformTradeSupportsGen1SameGeneration(t *testing.T) {
	saveA := gen1SaveForMobile(gen1RecordForMobile(0x99, 5))
	saveB := gen1SaveForMobile(gen1RecordForMobile(0xB0, 12))

	preview := PreviewTrade(saveA, saveB, 1, 1)
	if !preview.Supported {
		t.Fatalf("Gen 1 pair should be supported: %s", preview.Message)
	}

	result, err := PerformTrade(saveA, saveB, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	fileA, err := core.LoadSaveFromBytes(result.SaveA)
	if err != nil {
		t.Fatalf("traded Gen 1 save A no longer parses: %v", err)
	}
	partyA, err := fileA.GetParty()
	if err != nil {
		t.Fatal(err)
	}
	if partyA[0].GetLevel() != 12 {
		t.Fatalf("Gen 1 trade did not move the record: level %d, want 12", partyA[0].GetLevel())
	}
}

func TestPreviewTradeBlocksCrossGeneration(t *testing.T) {
	gen1 := gen1SaveForMobile(gen1RecordForMobile(0x99, 5))
	gen3 := saveWithParty(makeRecord(1, 4))
	preview := PreviewTrade(gen1, gen3, 1, 1)
	if preview.Supported {
		t.Fatal("cross-generation preview must not be runnable")
	}
	if preview.Message == "" {
		t.Fatal("cross-generation preview should explain why it is blocked")
	}
}

func TestPreviewTradeBlocksGenerationWithoutWriter(t *testing.T) {
	// A 512 KiB file is claimed by the Generation 4 parser, which has no writer.
	saveA := make([]byte, 524288)
	saveB := make([]byte, 524288)
	preview := PreviewTrade(saveA, saveB, 1, 1)
	if preview.Supported {
		t.Fatal("generation without a verified writer must not be runnable")
	}
	if !strings.Contains(preview.Message, "Gen 4") {
		t.Fatalf("preview should name the unsupported generation: %s", preview.Message)
	}
}

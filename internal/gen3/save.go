package gen3

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	SlotAOffset     = 0x0000
	SlotBOffset     = 0xE000
	SectionSize     = 4096
	SectionsPerSlot = 14
	Signature       = 0x08012025
)

var sectionDataSizes = map[uint16]int{
	0:  3884,
	1:  3968,
	2:  3968,
	3:  3968,
	4:  3848,
	5:  3968,
	6:  3968,
	7:  3968,
	8:  3968,
	9:  3968,
	10: 3968,
	11: 3968,
	12: 3968,
	13: 2000,
}

type SaveSlot struct {
	Index    uint32
	Sections map[uint16][]byte
	Valid    bool
}

type SaveFile struct {
	ActiveSlot *SaveSlot
}

func sectionChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+4 <= len(data); i += 4 {
		sum += binary.LittleEndian.Uint32(data[i:])
	}
	return uint16(sum>>16) + uint16(sum)
}

func LoadSave(path string) (*SaveFile, error) {
	fmt.Printf("--- Loading %s ---\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) != 131072 {
		return nil, fmt.Errorf("invalid save file size: %d bytes (expected 131072)", len(data))
	}

	slotA := parseAndPrintSlot("Slot A", data, SlotAOffset)
	slotB := parseAndPrintSlot("Slot B", data, SlotBOffset)

	var active *SaveSlot
	if slotA.Valid && slotB.Valid {
		if slotA.Index > slotB.Index {
			active = slotA
			fmt.Printf("Active slot is Slot A (Index: %d > %d)\n", slotA.Index, slotB.Index)
		} else {
			active = slotB
			fmt.Printf("Active slot is Slot B (Index: %d > %d)\n", slotB.Index, slotA.Index)
		}
	} else if slotA.Valid {
		active = slotA
		fmt.Println("Active slot is Slot A (Slot B invalid)")
	} else if slotB.Valid {
		active = slotB
		fmt.Println("Active slot is Slot B (Slot A invalid)")
	} else {
		return nil, fmt.Errorf("both save slots are corrupt")
	}

	return &SaveFile{ActiveSlot: active}, nil
}

func parseAndPrintSlot(name string, data []byte, offset int) *SaveSlot {
	fmt.Printf("%s:\n", name)
	slot := &SaveSlot{
		Sections: make(map[uint16][]byte),
		Valid:    true,
	}

	var maxSaveIndex uint32

	for i := 0; i < SectionsPerSlot; i++ {
		secOffset := offset + i*SectionSize
		sectionData := data[secOffset : secOffset+SectionSize]

		sectionID := binary.LittleEndian.Uint16(sectionData[0xFF4:])
		checksum := binary.LittleEndian.Uint16(sectionData[0xFF6:])
		signature := binary.LittleEndian.Uint32(sectionData[0xFF8:])
		saveIndex := binary.LittleEndian.Uint32(sectionData[0xFFC:])

		if saveIndex > maxSaveIndex && signature == Signature {
			maxSaveIndex = saveIndex
		}

		dataSize, ok := sectionDataSizes[sectionID]
		status := "PASS"

		if signature != Signature {
			status = fmt.Sprintf("FAIL (Bad Sig: 0x%08X)", signature)
			slot.Valid = false
		} else if !ok {
			status = fmt.Sprintf("FAIL (Unknown ID: %d)", sectionID)
			slot.Valid = false
		} else {
			actualChecksum := sectionChecksum(sectionData[:dataSize])
			if actualChecksum != checksum {
				status = fmt.Sprintf("FAIL (Bad Checksum: expected 0x%04X, got 0x%04X)", checksum, actualChecksum)
				slot.Valid = false
			}
		}

		fmt.Printf("  Section at offset 0x%04X: ID=%2d, SaveIndex=%5d, Status=%s\n", secOffset, sectionID, saveIndex, status)

		if status == "PASS" {
			slot.Sections[sectionID] = sectionData
		}
	}

	slot.Index = maxSaveIndex
	fmt.Printf("  -> Final Save Index: %d, Valid: %v\n", slot.Index, slot.Valid)
	return slot
}

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
	Index          uint32
	Sections       map[uint16][]byte
	SectionOffsets map[uint16]int
	Valid          bool
}

type SaveFile struct {
	ActiveSlot *SaveSlot
	RawData    []byte
}

func sectionChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+4 <= len(data); i += 4 {
		sum += binary.LittleEndian.Uint32(data[i:])
	}
	return uint16(sum>>16) + uint16(sum)
}

func LoadSave(path string) (*SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) != 131072 {
		return nil, fmt.Errorf("invalid save file size: %d bytes (expected 131072)", len(data))
	}

	slotA := parseAndPrintSlot(data, SlotAOffset)
	slotB := parseAndPrintSlot(data, SlotBOffset)

	var active *SaveSlot
	if slotA.Valid && slotB.Valid {
		if slotA.Index > slotB.Index {
			active = slotA
		} else {
			active = slotB
		}
	} else if slotA.Valid {
		active = slotA
	} else if slotB.Valid {
		active = slotB
	} else {
		return nil, fmt.Errorf("both save slots are corrupt")
	}

	return &SaveFile{ActiveSlot: active, RawData: data}, nil
}

func (sf *SaveFile) WriteToFile(path string) error {
	outData := make([]byte, len(sf.RawData))
	copy(outData, sf.RawData)

	for id, secData := range sf.ActiveSlot.Sections {
		offset := sf.ActiveSlot.SectionOffsets[id]

		dataSize := sectionDataSizes[id]
		newChecksum := sectionChecksum(secData[:dataSize])
		binary.LittleEndian.PutUint16(secData[0xFF6:], newChecksum)

		copy(outData[offset:offset+SectionSize], secData)
	}

	return os.WriteFile(path, outData, 0644)
}

func parseAndPrintSlot(data []byte, offset int) *SaveSlot {
	slot := &SaveSlot{
		Sections:       make(map[uint16][]byte),
		SectionOffsets: make(map[uint16]int),
		Valid:          true,
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
		status := true

		if signature != Signature || !ok {
			status = false
			slot.Valid = false
		} else {
			actualChecksum := sectionChecksum(sectionData[:dataSize])
			if actualChecksum != checksum {
				status = false
				slot.Valid = false
			}
		}

		if status {
			slot.Sections[sectionID] = sectionData
			slot.SectionOffsets[sectionID] = secOffset
		}
	}

	slot.Index = maxSaveIndex
	return slot
}

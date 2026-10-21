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

	// FRLG stores game code 1 at section 0 + 0xAC. RSE uses a different
	// value, so this is the strongest cheap discriminator available without
	// decoding every section.
	FRLGGameCode = 1
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

// LoadSaveFromBytes validates a full 128 KiB FRLG save. Both slots are parsed;
// the newest fully valid slot is selected. A slot is only accepted when all
// fourteen sections are present exactly once, every section signature and
// checksum matches, and all sections agree on the save index.
func LoadSaveFromBytes(data []byte) (*SaveFile, error) {
	if len(data) != 131072 {
		return nil, fmt.Errorf("invalid save file size: %d bytes (expected 131072)", len(data))
	}

	// Copy first so section slices cannot alias the caller's buffer.
	raw := append([]byte(nil), data...)
	slotA, errA := parseSlot(raw, SlotAOffset)
	slotB, errB := parseSlot(raw, SlotBOffset)

	var active *SaveSlot
	switch {
	case slotA != nil && slotB != nil:
		if slotA.Index > slotB.Index {
			active = slotA
		} else {
			active = slotB
		}
	case slotA != nil:
		active = slotA
	case slotB != nil:
		active = slotB
	default:
		return nil, fmt.Errorf("both save slots are corrupt (slot A: %v; slot B: %v)", errA, errB)
	}

	save := &SaveFile{ActiveSlot: active, RawData: raw}
	if err := save.Validate(); err != nil {
		return nil, err
	}
	return save, nil
}

func LoadSave(path string) (*SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadSaveFromBytes(data)
}

// Validate checks that the selected slot is a supported FRLG save with a
// readable party.
func (sf *SaveFile) Validate() error {
	if sf == nil || sf.ActiveSlot == nil {
		return fmt.Errorf("no active save slot")
	}
	if len(sf.RawData) != 131072 {
		return fmt.Errorf("raw save is %d bytes, expected 131072", len(sf.RawData))
	}
	if sf.ActiveSlot.Index == 0 {
		return fmt.Errorf("save index is zero; this is not a written save")
	}
	for id := uint16(0); id < SectionsPerSlot; id++ {
		if len(sf.ActiveSlot.Sections[id]) != SectionSize {
			return fmt.Errorf("section %d is missing", id)
		}
	}
	sec0 := sf.ActiveSlot.Sections[0]
	if len(sec0) < 0xB0 {
		return fmt.Errorf("section 0 is truncated")
	}
	if gameCode := binary.LittleEndian.Uint32(sec0[0xAC:]); gameCode != FRLGGameCode {
		return fmt.Errorf("not a FireRed/LeafGreen save: game code %d (expected %d)", gameCode, FRLGGameCode)
	}
	if _, err := GetParty(sf.ActiveSlot); err != nil {
		return fmt.Errorf("party data is invalid: %w", err)
	}
	return nil
}

func parseSlot(data []byte, offset int) (*SaveSlot, error) {
	slot := &SaveSlot{
		Sections:       make(map[uint16][]byte, SectionsPerSlot),
		SectionOffsets: make(map[uint16]int, SectionsPerSlot),
	}
	var saveIndex uint32
	haveIndex := false

	for i := 0; i < SectionsPerSlot; i++ {
		secOffset := offset + i*SectionSize
		if secOffset+SectionSize > len(data) {
			return nil, fmt.Errorf("section %d is outside the file", i)
		}
		sectionData := data[secOffset : secOffset+SectionSize]
		sectionID := binary.LittleEndian.Uint16(sectionData[0xFF4:])
		checksum := binary.LittleEndian.Uint16(sectionData[0xFF6:])
		signature := binary.LittleEndian.Uint32(sectionData[0xFF8:])
		index := binary.LittleEndian.Uint32(sectionData[0xFFC:])

		if signature != Signature {
			return nil, fmt.Errorf("section %d has signature %08X, expected %08X", i, signature, Signature)
		}
		if sectionID >= SectionsPerSlot {
			return nil, fmt.Errorf("section %d has invalid id %d", i, sectionID)
		}
		if _, exists := slot.Sections[sectionID]; exists {
			return nil, fmt.Errorf("duplicate section id %d", sectionID)
		}
		dataSize, ok := sectionDataSizes[sectionID]
		if !ok || dataSize > SectionSize {
			return nil, fmt.Errorf("section %d has unknown data size", sectionID)
		}
		if actual := sectionChecksum(sectionData[:dataSize]); actual != checksum {
			return nil, fmt.Errorf("section %d checksum mismatch: stored %04X, calculated %04X", sectionID, checksum, actual)
		}
		if !haveIndex {
			saveIndex = index
			haveIndex = true
		} else if index != saveIndex {
			return nil, fmt.Errorf("section %d save index %d does not match %d", sectionID, index, saveIndex)
		}
		slot.Sections[sectionID] = sectionData
		slot.SectionOffsets[sectionID] = secOffset
	}

	if len(slot.Sections) != SectionsPerSlot {
		return nil, fmt.Errorf("only %d of %d sections are valid", len(slot.Sections), SectionsPerSlot)
	}
	slot.Index = saveIndex
	slot.Valid = true
	return slot, nil
}

// WriteToBytes produces a complete save without mutating the in-memory
// sections. Checksums are computed from the current section contents.
func (sf *SaveFile) WriteToBytes() []byte {
	outData := append([]byte(nil), sf.RawData...)
	if sf.ActiveSlot == nil {
		return outData
	}
	for id, secData := range sf.ActiveSlot.Sections {
		offset, ok := sf.ActiveSlot.SectionOffsets[id]
		if !ok || len(secData) != SectionSize {
			continue
		}
		dataSize := sectionDataSizes[id]
		checksum := sectionChecksum(secData[:dataSize])
		// Copy section first, then write its checksum into the output only.
		copy(outData[offset:offset+SectionSize], secData)
		binary.LittleEndian.PutUint16(outData[offset+0xFF6:], checksum)
	}
	return outData
}

func (sf *SaveFile) WriteToFile(path string) error {
	return os.WriteFile(path, sf.WriteToBytes(), 0644)
}

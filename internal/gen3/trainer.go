package gen3

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type TrainerInfo struct {
	Name   string
	Gender byte
	TID    uint16
	SID    uint16
}

func GetTrainerInfo(slot *SaveSlot) (*TrainerInfo, error) {
	sec0, ok := slot.Sections[0]
	if !ok {
		return nil, fmt.Errorf("section 0 missing")
	}

	gameCode := binary.LittleEndian.Uint32(sec0[0xAC:])
	if gameCode != 1 {
		return nil, fmt.Errorf("invalid game code: %d (expected 1 for FRLG)", gameCode)
	}

	name := decodeGen3String(sec0[0x00:0x07])
	gender := sec0[0x08]
	tid := binary.LittleEndian.Uint16(sec0[0x0A:])
	sid := binary.LittleEndian.Uint16(sec0[0x0C:])

	return &TrainerInfo{
		Name:   name,
		Gender: gender,
		TID:    tid,
		SID:    sid,
	}, nil
}

func decodeGen3String(data []byte) string {
	var buf bytes.Buffer
	for _, b := range data {
		if b == 0xFF {
			break
		}
		if b == 0x00 {
			buf.WriteByte(' ')
		} else if b >= 0xA1 && b <= 0xAA {
			buf.WriteByte('0' + (b - 0xA1))
		} else if b >= 0xBB && b <= 0xD4 {
			buf.WriteByte('A' + (b - 0xBB))
		} else if b >= 0xD5 && b <= 0xEE {
			buf.WriteByte('a' + (b - 0xD5))
		} else {
			// fallback for unmapped characters
			buf.WriteByte('?')
		}
	}
	return buf.String()
}
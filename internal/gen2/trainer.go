package gen2

import "encoding/binary"

type TrainerInfo struct {
	Name string
	TID  uint16
}

func GetTrainerInfo(data []byte) *TrainerInfo {
	if len(data) < 0x2016 {
		return nil
	}
	return &TrainerInfo{
		Name: decodeGen2String(data[0x200B : 0x200B+NameLength]),
		TID:  binary.BigEndian.Uint16(data[0x2008:]),
	}
}

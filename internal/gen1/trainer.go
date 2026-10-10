package gen1

type TrainerInfo struct {
	Name string
	TID  uint16
}

func GetTrainerInfo(data []byte) *TrainerInfo {
	// Gen 1 trainer name is at 0x2598
	name := decodeGen1String(data[0x2598 : 0x2598+11])
	tid := uint16(data[0x2598+11])<<8 | uint16(data[0x2598+12])
	return &TrainerInfo{Name: name, TID: tid}
}

package gen2

type TrainerInfo struct {
	Name string
	TID  uint16
}

func GetTrainerInfo(data []byte) *TrainerInfo {
	return &TrainerInfo{Name: "Gen2Player", TID: 12345}
}

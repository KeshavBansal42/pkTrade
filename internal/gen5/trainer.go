package gen5

type TrainerInfo struct {
	Name string
	TID  uint16
	SID  uint16
}

func GetTrainerInfo(data []byte) *TrainerInfo {
	return &TrainerInfo{Name: "Gen5Player", TID: 11111, SID: 22222}
}

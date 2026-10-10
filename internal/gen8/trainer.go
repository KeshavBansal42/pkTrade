package gen8

type TrainerInfo struct {
	Name string
	TID  uint16
	SID  uint16
}

func GetTrainerInfo(data []byte) *TrainerInfo {
	return &TrainerInfo{Name: "Gen8Player", TID: 11111, SID: 22222}
}

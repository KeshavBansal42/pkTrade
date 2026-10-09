package mobile

import (
	"encoding/json"
	"fmt"
	"pktrade/internal/gen3"
)

type PokemonResponse struct {
	Index    int    `json:"index"`
	Species  uint16 `json:"species"`
	Level    byte   `json:"level"`
	Nickname string `json:"nickname"`
	OTName   string `json:"otName"`
	OTID     uint16 `json:"otId"`
}

type PartyResponse struct {
	TrainerName string            `json:"trainerName"`
	TID         uint16            `json:"tid"`
	SID         uint16            `json:"sid"`
	Party       []PokemonResponse `json:"party"`
}

// ParsePartyJSON parses a save file byte array and returns the Party Pokemon as a JSON string.
// This makes it extremely easy to render the UI natively in Kotlin/Flutter.
func ParsePartyJSON(saveData []byte) (string, error) {
	saveFile, err := gen3.LoadSaveFromBytes(saveData)
	if err != nil {
		return "", err
	}

	trainer, err := gen3.GetTrainerInfo(saveFile.ActiveSlot)
	if err != nil {
		return "", err
	}

	party, err := gen3.GetParty(saveFile.ActiveSlot)
	if err != nil {
		return "", err
	}

	res := PartyResponse{
		TrainerName: trainer.Name,
		TID:         trainer.TID,
		SID:         trainer.SID,
		Party:       make([]PokemonResponse, len(party)),
	}

	for i, p := range party {
		res.Party[i] = PokemonResponse{
			Index:    i + 1,
			Species:  p.Species,
			Level:    p.Level,
			Nickname: p.Nickname,
			OTName:   p.OTName,
			OTID:     uint16(p.OTID & 0xFFFF),
		}
	}

	jsonBytes, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// TradeResult encapsulates the modified byte arrays to return to Android.
type TradeResult struct {
	SaveA []byte
	SaveB []byte
}

// PerformTrade executes the raw byte swap and evolution logic entirely in memory.
// It takes 1-indexed party indices (idxA, idxB).
func PerformTrade(saveAData []byte, saveBData []byte, idxA int, idxB int) (*TradeResult, error) {
	saveFileA, errA := gen3.LoadSaveFromBytes(saveAData)
	saveFileB, errB := gen3.LoadSaveFromBytes(saveBData)
	if errA != nil {
		return nil, fmt.Errorf("failed to load Save A: %w", errA)
	}
	if errB != nil {
		return nil, fmt.Errorf("failed to load Save B: %w", errB)
	}

	if idxA < 1 || idxA > 6 || idxB < 1 || idxB > 6 {
		return nil, fmt.Errorf("invalid party index: must be between 1 and 6")
	}

	offsetA := 0x38 + (idxA-1)*100
	offsetB := 0x38 + (idxB-1)*100

	sec1A := saveFileA.ActiveSlot.Sections[1]
	sec1B := saveFileB.ActiveSlot.Sections[1]

	// 100-byte raw swap
	temp := make([]byte, 100)
	copy(temp, sec1A[offsetA:offsetA+100])
	copy(sec1A[offsetA:offsetA+100], sec1B[offsetB:offsetB+100])
	copy(sec1B[offsetB:offsetB+100], temp)

	// Process Evolutions
	newA, evolvedA, err := gen3.ProcessTradeEvolution(sec1A[offsetA : offsetA+100])
	if err == nil && evolvedA {
		copy(sec1A[offsetA:offsetA+100], newA)
	}

	newB, evolvedB, err := gen3.ProcessTradeEvolution(sec1B[offsetB : offsetB+100])
	if err == nil && evolvedB {
		copy(sec1B[offsetB:offsetB+100], newB)
	}

	return &TradeResult{
		SaveA: saveFileA.WriteToBytes(),
		SaveB: saveFileB.WriteToBytes(),
	}, nil
}

package mobile

import (
	"encoding/json"
	"fmt"
	"pktrade/internal/core"
	_ "pktrade/internal/gen1"
	_ "pktrade/internal/gen2"
	_ "pktrade/internal/gen3"
	_ "pktrade/internal/gen4"
	_ "pktrade/internal/gen5"
	_ "pktrade/internal/gen6"
	_ "pktrade/internal/gen7"
	_ "pktrade/internal/gen8"
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
	Generation  int               `json:"generation"`
	TrainerName string            `json:"trainerName"`
	TID         uint16            `json:"tid"`
	SID         uint16            `json:"sid"`
	Party       []PokemonResponse `json:"party"`
}

// ParsePartyJSON parses a save file byte array and returns the Party Pokemon as a JSON string.
// This makes it extremely easy to render the UI natively in Kotlin/Flutter.
func ParsePartyJSON(saveData []byte) (string, error) {
	saveFile, err := core.LoadSaveFromBytes(saveData)
	if err != nil {
		return "", err
	}

	party, err := saveFile.GetParty()
	if err != nil {
		return "", err
	}

	gen := saveFile.Generation()

	res := PartyResponse{
		Generation:  gen,
		TrainerName: "UniversalTrainer",
		TID:         0,
		SID:         0,
		Party:       make([]PokemonResponse, len(party)),
	}

	for i, p := range party {
		res.Party[i] = PokemonResponse{
			Index:    i + 1,
			Species:  p.GetSpecies(),
			Level:    p.GetLevel(),
			Nickname: p.GetNickname(),
			OTName:   p.GetTrainerName(),
			OTID:     uint16(p.GetTrainerID() & 0xFFFF),
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
	saveFileA, errA := core.LoadSaveFromBytes(saveAData)
	saveFileB, errB := core.LoadSaveFromBytes(saveBData)
	if errA != nil {
		return nil, fmt.Errorf("failed to load Save A: %w", errA)
	}
	if errB != nil {
		return nil, fmt.Errorf("failed to load Save B: %w", errB)
	}

	if idxA < 1 || idxA > 6 || idxB < 1 || idxB > 6 {
		return nil, fmt.Errorf("invalid party index: must be between 1 and 6")
	}

	partyA, _ := saveFileA.GetParty()
	partyB, _ := saveFileB.GetParty()

	temp := partyA[idxA-1]
	partyA[idxA-1] = partyB[idxB-1]
	partyB[idxB-1] = temp

	saveFileA.InjectParty(partyA)
	saveFileB.InjectParty(partyB)

	return &TradeResult{
		SaveA: saveFileA.WriteToBytes(),
		SaveB: saveFileB.WriteToBytes(),
	}, nil
}

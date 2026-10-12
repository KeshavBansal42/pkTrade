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
	BallID   *byte  `json:"ballId,omitempty"`
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
		response := PokemonResponse{
			Index:    i + 1,
			Species:  p.GetSpecies(),
			Level:    p.GetLevel(),
			Nickname: p.GetNickname(),
			OTName:   p.GetTrainerName(),
			OTID:     uint16(p.GetTrainerID() & 0xFFFF),
		}
		if ball, ok := p.(core.CaptureBall); ok {
			if id, known := ball.GetCaptureBall(); known {
				ballID := id
				response.BallID = &ballID
			}
		}
		res.Party[i] = response
	}

	jsonBytes, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// TradePreview describes whether the selected pair can be exchanged without
// dropping or fabricating record data.
type TradePreview struct {
	Supported bool     `json:"supported"`
	Lossy     bool     `json:"lossy"`
	Message   string   `json:"message"`
	Losses    []string `json:"losses"`
}

func PreviewTrade(saveAData []byte, saveBData []byte, idxA int, idxB int) *TradePreview {
	saveA, err := core.LoadSaveFromBytes(saveAData)
	if err != nil {
		return &TradePreview{Message: fmt.Sprintf("Save A could not be read: %v", err)}
	}
	saveB, err := core.LoadSaveFromBytes(saveBData)
	if err != nil {
		return &TradePreview{Message: fmt.Sprintf("Save B could not be read: %v", err)}
	}
	genA, genB := saveA.Generation(), saveB.Generation()
	if genA != genB {
		return &TradePreview{
			Message: fmt.Sprintf("Cross-generation trades are not supported (Gen %d into Gen %d). Both saves must be the same generation.", genA, genB),
		}
	}
	if !core.SupportsTrade(genA) {
		return &TradePreview{
			Message: fmt.Sprintf("Gen %d does not have a complete, verified party writer yet. No files will be changed.", genA),
		}
	}

	partyA, err := saveA.GetParty()
	if err != nil {
		return &TradePreview{Message: fmt.Sprintf("Save A party could not be read: %v", err)}
	}
	partyB, err := saveB.GetParty()
	if err != nil {
		return &TradePreview{Message: fmt.Sprintf("Save B party could not be read: %v", err)}
	}
	if idxA < 1 || idxA > len(partyA) || idxB < 1 || idxB > len(partyB) {
		return &TradePreview{Message: "Select an existing Pokémon from both parties."}
	}
	for _, pokemon := range append(append([]core.Pokemon(nil), partyA...), partyB...) {
		if !pokemon.IsChecksumValid() {
			return &TradePreview{Message: "A party record has an invalid checksum. No files will be changed."}
		}
	}
	return &TradePreview{
		Supported: true,
		Message:   fmt.Sprintf("Both are valid Gen %d saves. Complete party records will be exchanged byte-for-byte.", genA),
	}
}

// PreviewTradeJSON returns the transfer preview as JSON for the Android bridge.
func PreviewTradeJSON(saveAData []byte, saveBData []byte, idxA int, idxB int) string {
	preview := PreviewTrade(saveAData, saveBData, idxA, idxB)
	encoded, err := json.Marshal(preview)
	if err != nil {
		return `{"supported":false,"message":"Transfer preview could not be created."}`
	}
	return string(encoded)
}

// TradeResult encapsulates the modified byte arrays to return to Android.
type TradeResult struct {
	SaveA []byte
	SaveB []byte
}

// PerformTrade executes the raw byte swap and evolution logic entirely in memory.
// It takes 1-indexed party indices (idxA, idxB).
func PerformTrade(saveAData []byte, saveBData []byte, idxA int, idxB int) (*TradeResult, error) {
	preview := PreviewTrade(saveAData, saveBData, idxA, idxB)
	if !preview.Supported {
		return nil, fmt.Errorf("%s", preview.Message)
	}
	saveAData = append([]byte(nil), saveAData...)
	saveBData = append([]byte(nil), saveBData...)

	saveFileA, err := core.LoadSaveFromBytes(saveAData)
	if err != nil {
		return nil, fmt.Errorf("failed to load Save A: %w", err)
	}
	saveFileB, err := core.LoadSaveFromBytes(saveBData)
	if err != nil {
		return nil, fmt.Errorf("failed to load Save B: %w", err)
	}
	if saveFileA.Generation() != saveFileB.Generation() {
		return nil, fmt.Errorf("cross-generation trades are not supported")
	}

	partyA, err := saveFileA.GetParty()
	if err != nil {
		return nil, fmt.Errorf("failed to read Save A party: %w", err)
	}
	partyB, err := saveFileB.GetParty()
	if err != nil {
		return nil, fmt.Errorf("failed to read Save B party: %w", err)
	}
	if idxA < 1 || idxA > len(partyA) || idxB < 1 || idxB > len(partyB) {
		return nil, fmt.Errorf("selected party index does not exist")
	}
	for _, pokemon := range append(append([]core.Pokemon(nil), partyA...), partyB...) {
		if !pokemon.IsChecksumValid() {
			return nil, fmt.Errorf("trade refused because a party record has an invalid checksum")
		}
	}

	partyA[idxA-1], partyB[idxB-1] = partyB[idxB-1], partyA[idxA-1]
	if err := saveFileA.InjectParty(partyA); err != nil {
		return nil, fmt.Errorf("failed to update Save A party: %w", err)
	}
	if err := saveFileB.InjectParty(partyB); err != nil {
		return nil, fmt.Errorf("failed to update Save B party: %w", err)
	}

	return &TradeResult{
		SaveA: saveFileA.WriteToBytes(),
		SaveB: saveFileB.WriteToBytes(),
	}, nil
}

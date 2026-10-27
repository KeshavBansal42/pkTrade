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
	_ "pktrade/internal/xgen"
)

type PokemonResponse struct {
	Index    int    `json:"index"`
	Species  uint16 `json:"species"`
	National uint16 `json:"national,omitempty"`
	BallID   *byte  `json:"ballId,omitempty"`
	Level    byte   `json:"level"`
	Nickname string `json:"nickname"`
	OTName   string `json:"otName"`
	OTID     uint16 `json:"otId"`
	HeldItem uint16 `json:"heldItem,omitempty"`
	Slot     string `json:"slot,omitempty"`
	IsEgg    bool   `json:"isEgg,omitempty"`
	Checksum bool   `json:"checksumValid"`
}

type PartyResponse struct {
	Generation  int               `json:"generation"`
	Game        string            `json:"game,omitempty"`
	Family      string            `json:"family,omitempty"`
	TrainerName string            `json:"trainerName"`
	TID         uint16            `json:"tid"`
	SID         uint16            `json:"sid"`
	Party       []PokemonResponse `json:"party"`
}

type SlotEntryResponse struct {
	Slot     string `json:"slot"`
	Empty    bool   `json:"empty"`
	Index    int    `json:"index,omitempty"`
	Species  uint16 `json:"species,omitempty"`
	National uint16 `json:"national,omitempty"`
	Level    byte   `json:"level,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	OTName   string `json:"otName,omitempty"`
	OTID     uint16 `json:"otId,omitempty"`
	HeldItem uint16 `json:"heldItem,omitempty"`
	BallID   *byte  `json:"ballId,omitempty"`
}

// ParsePartyJSON parses a save file byte array and returns the party as JSON.
// Trainer fields are read from the save when the generation can provide them;
// older party-only paths fall back to the first Pokémon's OT data.
func ParsePartyJSON(saveData []byte) (string, error) {
	saveFile, err := core.LoadSaveFromBytes(saveData)
	if err != nil {
		return "", err
	}
	party, err := saveFile.GetParty()
	if err != nil {
		return "", err
	}

	res := PartyResponse{Generation: saveFile.Generation(), Party: make([]PokemonResponse, 0, len(party))}
	if info, ok := saveFile.(core.InfoProvider); ok {
		game := info.Info()
		res.Game, res.Family = game.Game, game.Family
	}
	if p, ok := saveFile.(core.TrainerProvider); ok {
		if trainer, err := p.TrainerInfo(); err == nil {
			res.TrainerName, res.TID, res.SID = trainer.Name, trainer.TID, trainer.SID
		}
	}

	for i, p := range party {
		response := PokemonResponse{
			Index:    i + 1,
			Species:  p.GetSpecies(),
			Level:    p.GetLevel(),
			Nickname: p.GetNickname(),
			OTName:   p.GetTrainerName(),
			OTID:     uint16(p.GetTrainerID() & 0xFFFF),
			HeldItem: p.GetHeldItem(),
			Slot:     fmt.Sprintf("party:%d", i+1),
			Checksum: p.IsChecksumValid(),
		}
		if n, ok := p.(interface{ NationalID() uint16 }); ok {
			response.National = n.NationalID()
		}
		if ball, ok := p.(core.CaptureBall); ok {
			if id, known := ball.GetCaptureBall(); known {
				ballID := id
				response.BallID = &ballID
			}
		}
		res.Party = append(res.Party, response)
	}

	// If the save cannot report a trainer, use the first record's OT identity.
	if res.TrainerName == "" && len(party) > 0 {
		res.TrainerName = party[0].GetTrainerName()
		res.TID = uint16(party[0].GetTrainerID() & 0xFFFF)
	}
	if res.TrainerName == "" {
		res.TrainerName = "TRAINER"
	}
	jsonBytes, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// PreviewTrade is the original party-index entry point. Indices are 1-based.
func PreviewTrade(saveAData []byte, saveBData []byte, idxA int, idxB int) *core.TradePreview {
	if idxA < 1 || idxB < 1 {
		return &core.TradePreview{
			Supported: false,
			Losses:    []string{},
			Message:   "Party slots are one-based; select a slot between 1 and 6.",
		}
	}
	saveA, err := core.LoadSaveFromBytes(saveAData)
	if err != nil {
		return &core.TradePreview{Supported: false, Losses: []string{}, Message: fmt.Sprintf("Save A could not be read: %v", err)}
	}
	saveB, err := core.LoadSaveFromBytes(saveBData)
	if err != nil {
		return &core.TradePreview{Supported: false, Losses: []string{}, Message: fmt.Sprintf("Save B could not be read: %v", err)}
	}
	return core.PreviewPartyTrade(saveA, saveB, idxA, idxB)
}

// PreviewSlotTrade accepts the shared party:N / box:B/S slot syntax.
func PreviewSlotTrade(saveAData []byte, saveBData []byte, slotA string, slotB string) *core.TradePreview {
	refA, err := core.ParseSlot(slotA)
	if err != nil {
		return &core.TradePreview{Supported: false, Losses: []string{}, Message: fmt.Sprintf("Save A slot: %v", err)}
	}
	refB, err := core.ParseSlot(slotB)
	if err != nil {
		return &core.TradePreview{Supported: false, Losses: []string{}, Message: fmt.Sprintf("Save B slot: %v", err)}
	}
	saveA, err := core.LoadSaveFromBytes(saveAData)
	if err != nil {
		return &core.TradePreview{Supported: false, Losses: []string{}, Message: fmt.Sprintf("Save A could not be read: %v", err)}
	}
	saveB, err := core.LoadSaveFromBytes(saveBData)
	if err != nil {
		return &core.TradePreview{Supported: false, Losses: []string{}, Message: fmt.Sprintf("Save B could not be read: %v", err)}
	}
	return core.PreviewTrade(saveA, saveB, refA, refB)
}

// PreviewTradeJSON returns the transfer preview as JSON for the Android bridge.
func PreviewTradeJSON(saveAData []byte, saveBData []byte, idxA int, idxB int) string {
	return marshalPreview(PreviewTrade(saveAData, saveBData, idxA, idxB))
}

// PreviewSlotTradeJSON is the slot-syntax variant for box-aware callers.
func PreviewSlotTradeJSON(saveAData []byte, saveBData []byte, slotA string, slotB string) string {
	return marshalPreview(PreviewSlotTrade(saveAData, saveBData, slotA, slotB))
}

func marshalPreview(preview *core.TradePreview) string {
	encoded, err := json.Marshal(preview)
	if err != nil {
		return `{"supported":false,"lossy":false,"message":"Transfer preview could not be created.","losses":[]}`
	}
	return string(encoded)
}

// TradeResult encapsulates the modified byte arrays to return to Android.
type TradeResult struct {
	SaveA      []byte
	SaveB      []byte
	Evolutions []core.Evolution
}

// PerformTrade executes a party trade. It takes 1-indexed party indices.
func PerformTrade(saveAData []byte, saveBData []byte, idxA int, idxB int) (*TradeResult, error) {
	if idxA < 1 || idxB < 1 {
		return nil, fmt.Errorf("party slots are one-based")
	}
	return performSlotTrade(saveAData, saveBData, core.Party(idxA), core.Party(idxB))
}

// PerformSlotTrade executes a party or box trade using party:N / box:B/S.
func PerformSlotTrade(saveAData []byte, saveBData []byte, slotA string, slotB string) (*TradeResult, error) {
	refA, err := core.ParseSlot(slotA)
	if err != nil {
		return nil, err
	}
	refB, err := core.ParseSlot(slotB)
	if err != nil {
		return nil, err
	}
	return performSlotTrade(saveAData, saveBData, refA, refB)
}

func performSlotTrade(saveAData []byte, saveBData []byte, refA, refB core.SlotRef) (*TradeResult, error) {
	// Copy both inputs so a rejected or failed trade can never mutate the
	// caller's slices (some legacy parsers write in place).
	a := append([]byte(nil), saveAData...)
	b := append([]byte(nil), saveBData...)
	saveA, err := core.LoadSaveFromBytes(a)
	if err != nil {
		return nil, fmt.Errorf("Save A could not be read: %w", err)
	}
	saveB, err := core.LoadSaveFromBytes(b)
	if err != nil {
		return nil, fmt.Errorf("Save B could not be read: %w", err)
	}
	result, err := core.ExecuteTrade(saveA, saveB, refA, refB)
	if err != nil {
		return nil, err
	}
	return &TradeResult{SaveA: result.SaveA, SaveB: result.SaveB, Evolutions: result.Evolutions}, nil
}

// SlotsResponse describes every slot of a save: the party and, when the
// format has them, the boxes.
type SlotsResponse struct {
	Generation  int                 `json:"generation"`
	BoxCount    int                 `json:"boxCount"`
	BoxCapacity int                 `json:"boxCapacity"`
	Party       []PokemonResponse   `json:"party"`
	Slots       []SlotEntryResponse `json:"slots"`
}

// ListSlotsJSON lists every occupied slot in a save, together with the party
// and the box geometry, so a caller can offer the boxes to the player.
func ListSlotsJSON(saveData []byte, includeEmpty bool) (string, error) {
	save, err := core.LoadSaveFromBytes(saveData)
	if err != nil {
		return "", err
	}
	out := SlotsResponse{Generation: save.Generation(), Party: make([]PokemonResponse, 0, 6)}

	party, err := save.GetParty()
	if err != nil {
		return "", err
	}
	for i, p := range party {
		entry := PokemonResponse{
			Index:    i + 1,
			Species:  p.GetSpecies(),
			Level:    p.GetLevel(),
			Nickname: p.GetNickname(),
			OTName:   p.GetTrainerName(),
			OTID:     uint16(p.GetTrainerID() & 0xFFFF),
			HeldItem: p.GetHeldItem(),
			Slot:     fmt.Sprintf("party:%d", i+1),
			Checksum: p.IsChecksumValid(),
		}
		if n, ok := p.(interface{ NationalID() uint16 }); ok {
			entry.National = n.NationalID()
		}
		if ball, ok := p.(core.CaptureBall); ok {
			if id, known := ball.GetCaptureBall(); known {
				ballID := id
				entry.BallID = &ballID
			}
		}
		out.Party = append(out.Party, entry)
	}

	store, ok := save.(core.Storage)
	if !ok {
		encoded, err := json.Marshal(out)
		return string(encoded), err
	}
	out.BoxCount = store.BoxCount()
	out.BoxCapacity = store.BoxCapacity()

	entries, err := store.ListSlots(includeEmpty)
	if err != nil {
		return "", err
	}
	out.Slots = make([]SlotEntryResponse, 0, len(entries))
	for _, entry := range entries {
		item := SlotEntryResponse{Slot: entry.Ref.String(), Empty: entry.Empty}
		if entry.Pokemon != nil {
			p := entry.Pokemon
			item.Species = p.GetSpecies()
			item.Level = p.GetLevel()
			item.Nickname = p.GetNickname()
			item.OTName = p.GetTrainerName()
			item.OTID = uint16(p.GetTrainerID() & 0xFFFF)
			item.HeldItem = p.GetHeldItem()
			if n, ok := p.(interface{ NationalID() uint16 }); ok {
				item.National = n.NationalID()
			}
			if ball, ok := p.(core.CaptureBall); ok {
				if id, known := ball.GetCaptureBall(); known {
					ballID := id
					item.BallID = &ballID
				}
			}
		}
		out.Slots = append(out.Slots, item)
	}
	encoded, err := json.Marshal(out)
	return string(encoded), err
}

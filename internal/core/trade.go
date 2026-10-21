package core

import (
	"bytes"
	"fmt"
)

// TradePreview is the result of checking a proposed trade without changing
// anything. Supported is the only field callers may use to decide whether a
// trade is safe to execute.
type TradePreview struct {
	Supported  bool        `json:"supported"`
	Lossy      bool        `json:"lossy"`
	Message    string      `json:"message"`
	Losses     []string    `json:"losses"`
	Evolutions []Evolution `json:"evolutions,omitempty"`
	Warnings   []string    `json:"warnings,omitempty"`
	Source     string      `json:"source,omitempty"`
	Target     string      `json:"target,omitempty"`
}

// TradeResult contains the two rewritten saves and the evolutions that were
// applied. The byte slices are complete replacement files.
type TradeResult struct {
	SaveA      []byte      `json:"-"`
	SaveB      []byte      `json:"-"`
	Evolutions []Evolution `json:"evolutions,omitempty"`
	Warnings   []string    `json:"warnings,omitempty"`
}

func unsupported(format string, args ...any) *TradePreview {
	return &TradePreview{
		Supported: false,
		Lossy:     false,
		Message:   fmt.Sprintf(format, args...),
		Losses:    []string{},
	}
}

func infoOf(save SaveFile) GameInfo {
	if p, ok := save.(InfoProvider); ok {
		return p.Info()
	}
	return GameInfo{Generation: save.Generation(), Family: fmt.Sprintf("gen%d", save.Generation())}
}

func compatibilityOf(save SaveFile) (CompatibilityKey, error) {
	p, ok := save.(CompatibilityProvider)
	if !ok {
		return CompatibilityKey{}, fmt.Errorf("Gen %d saves do not declare a record format and cannot be traded", save.Generation())
	}
	return p.CompatibilityKey(), nil
}

func validateSave(save SaveFile) error {
	if v, ok := save.(Validator); ok {
		return v.Validate()
	}
	return nil
}

func slotsEqual(a, b []byte) bool {
	return bytes.Equal(a, b)
}

// PreviewTrade checks whether two loaded saves can exchange the selected
// slots. It is deliberately side-effect free: no save bytes are modified.
func PreviewTrade(saveA, saveB SaveFile, refA, refB SlotRef) *TradePreview {
	if err := refA.Validate(); err != nil {
		return unsupported("Save A slot is invalid: %v", err)
	}
	if err := refB.Validate(); err != nil {
		return unsupported("Save B slot is invalid: %v", err)
	}
	if err := validateSave(saveA); err != nil {
		return unsupported("Save A is not valid: %v", err)
	}
	if err := validateSave(saveB); err != nil {
		return unsupported("Save B is not valid: %v", err)
	}

	keyA, err := compatibilityOf(saveA)
	if err != nil {
		return unsupported("%v", err)
	}
	keyB, err := compatibilityOf(saveB)
	if err != nil {
		return unsupported("%v", err)
	}
	if keyA != keyB {
		infoA, infoB := infoOf(saveA), infoOf(saveB)
		return unsupported(
			"These saves are not compatible: %s (%s) cannot trade with %s (%s). "+
				"Trades require the same generation and the same save format; "+
				"for example FireRed/LeafGreen, Ruby/Sapphire/Emerald, and "+
				"Diamond/Pearl/Platinum/HeartGold/SoulSilver are separate formats.",
			infoA.Game, infoA.Family, infoB.Game, infoB.Family,
		)
	}

	if bytes.Equal(saveA.WriteToBytes(), saveB.WriteToBytes()) {
		return unsupported("Save A and Save B contain the same save data. Choose two different save files.")
	}

	capsA := capabilitiesOf(saveA)
	capsB := capabilitiesOf(saveB)
	if refA.Kind == BoxSlot && (!capsA.Boxes || !capsB.Boxes) {
		return unsupported("One of these saves does not support box trades. Only party trades are available.")
	}
	if (refA.Kind == PartySlot || refB.Kind == PartySlot) && (!capsA.Party || !capsB.Party) {
		return unsupported("One of these saves does not support party trades.")
	}

	storeA, storeB := storageFor(saveA), storageFor(saveB)
	if err := checkSlotCapacity(storeA, refA, "Save A"); err != nil {
		return unsupported("%v", err)
	}
	if err := checkSlotCapacity(storeB, refB, "Save B"); err != nil {
		return unsupported("%v", err)
	}

	pokemonA, err := storeA.ReadSlot(refA)
	if err != nil {
		return unsupported("Save A %s could not be read: %v", refA, err)
	}
	pokemonB, err := storeB.ReadSlot(refB)
	if err != nil {
		return unsupported("Save B %s could not be read: %v", refB, err)
	}
	if pokemonA == nil || pokemonA.GetSpecies() == 0 {
		return unsupported("Save A %s is empty.", refA)
	}
	if pokemonB == nil || pokemonB.GetSpecies() == 0 {
		return unsupported("Save B %s is empty.", refB)
	}
	if !pokemonA.IsChecksumValid() {
		return unsupported("Save A %s has an invalid record checksum. No files will be changed.", refA)
	}
	if !pokemonB.IsChecksumValid() {
		return unsupported("Save B %s has an invalid record checksum. No files will be changed.", refB)
	}
	if reason := tradeBlockReason(pokemonA); reason != "" {
		return unsupported("Save A %s cannot be traded: %s", refA, reason)
	}
	if reason := tradeBlockReason(pokemonB); reason != "" {
		return unsupported("Save B %s cannot be traded: %s", refB, reason)
	}

	preview := &TradePreview{
		Supported: true,
		Lossy:     false,
		Losses:    []string{},
		Source:    refA.String(),
		Target:    refB.String(),
	}

	if hooks, ok := saveA.(TradeHooks); ok {
		_, evolutions, err := hooks.FixupReceived(pokemonB)
		if err != nil {
			return unsupported("Save A cannot receive %s: %v", pokemonB.GetNickname(), err)
		}
		preview.Evolutions = append(preview.Evolutions, evolutions...)
	}
	if hooks, ok := saveB.(TradeHooks); ok {
		_, evolutions, err := hooks.FixupReceived(pokemonA)
		if err != nil {
			return unsupported("Save B cannot receive %s: %v", pokemonA.GetNickname(), err)
		}
		preview.Evolutions = append(preview.Evolutions, evolutions...)
	}

	preview.Message = fmt.Sprintf(
		"Ready to exchange Save A %s (%s, Lv. %d) with Save B %s (%s, Lv. %d).",
		refA, pokemonA.GetNickname(), pokemonA.GetLevel(),
		refB, pokemonB.GetNickname(), pokemonB.GetLevel(),
	)
	if len(preview.Evolutions) > 0 {
		names := make([]string, 0, len(preview.Evolutions))
		for _, evo := range preview.Evolutions {
			names = append(names, fmt.Sprintf("%s -> %s", evo.FromName, evo.ToName))
		}
		preview.Message += " Evolution: " + joinNames(names) + "."
	}
	return preview
}

func capabilitiesOf(save SaveFile) Capabilities {
	if p, ok := save.(CapabilityProvider); ok {
		return p.Capabilities()
	}
	return Capabilities{Party: true}
}

func checkSlotCapacity(store Storage, ref SlotRef, label string) error {
	switch ref.Kind {
	case PartySlot:
		if ref.Party > store.PartyCapacity() {
			return fmt.Errorf("%s party slot %d is outside the %d-slot party", label, ref.Party, store.PartyCapacity())
		}
	case BoxSlot:
		if store.BoxCount() == 0 {
			return fmt.Errorf("%s does not support box trades", label)
		}
		if ref.Box > store.BoxCount() {
			return fmt.Errorf("%s box %d is outside the %d available boxes", label, ref.Box, store.BoxCount())
		}
		if ref.Slot > store.BoxCapacity() {
			return fmt.Errorf("%s box slot %d is outside the %d slots per box", label, ref.Slot, store.BoxCapacity())
		}
	}
	return nil
}

func tradeBlockReason(pokemon Pokemon) string {
	if p, ok := pokemon.(TradeLegality); ok {
		return p.TradeBlockReason()
	}
	return ""
}

func joinNames(names []string) string {
	out := ""
	for i, name := range names {
		if i > 0 {
			if i == len(names)-1 {
				out += " and "
			} else {
				out += ", "
			}
		}
		out += name
	}
	return out
}

// ExecuteTrade performs the exchange after running the full preview. The
// caller's save objects are treated as owned by this function; the returned
// byte slices are complete files and are also applied to the objects.
func ExecuteTrade(saveA, saveB SaveFile, refA, refB SlotRef) (*TradeResult, error) {
	preview := PreviewTrade(saveA, saveB, refA, refB)
	if !preview.Supported {
		return nil, fmt.Errorf("%s", preview.Message)
	}

	originalA := append([]byte(nil), saveA.WriteToBytes()...)
	originalB := append([]byte(nil), saveB.WriteToBytes()...)

	storeA, storeB := storageFor(saveA), storageFor(saveB)
	pokemonA, err := storeA.ReadSlot(refA)
	if err != nil {
		return nil, fmt.Errorf("failed to read Save A %s: %w", refA, err)
	}
	pokemonB, err := storeB.ReadSlot(refB)
	if err != nil {
		return nil, fmt.Errorf("failed to read Save B %s: %w", refB, err)
	}

	incomingToA, incomingToB := pokemonB, pokemonA
	var evolutions []Evolution
	if hooks, ok := saveA.(TradeHooks); ok {
		incomingToA, evolutions, err = hooks.FixupReceived(pokemonB)
		if err != nil {
			return nil, fmt.Errorf("Save A cannot receive %s: %w", pokemonB.GetNickname(), err)
		}
	}
	var evolutionsB []Evolution
	if hooks, ok := saveB.(TradeHooks); ok {
		incomingToB, evolutionsB, err = hooks.FixupReceived(pokemonA)
		if err != nil {
			return nil, fmt.Errorf("Save B cannot receive %s: %w", pokemonA.GetNickname(), err)
		}
	}
	evolutions = append(evolutions, evolutionsB...)

	if err := storeA.WriteSlot(refA, incomingToA); err != nil {
		return nil, fmt.Errorf("failed to write Save A %s: %w", refA, err)
	}
	if err := storeB.WriteSlot(refB, incomingToB); err != nil {
		return nil, fmt.Errorf("failed to write Save B %s: %w", refB, err)
	}
	if hooks, ok := saveA.(TradeHooks); ok {
		if err := hooks.MarkDex(incomingToA); err != nil {
			return nil, fmt.Errorf("failed to update Save A Pokédex: %w", err)
		}
	}
	if hooks, ok := saveB.(TradeHooks); ok {
		if err := hooks.MarkDex(incomingToB); err != nil {
			return nil, fmt.Errorf("failed to update Save B Pokédex: %w", err)
		}
	}

	writtenA := saveA.WriteToBytes()
	writtenB := saveB.WriteToBytes()

	if v, ok := saveA.(OutputVerifier); ok {
		if err := v.VerifyOutput(originalA, writtenA); err != nil {
			return nil, fmt.Errorf("Save A integrity check failed: %w", err)
		}
	}
	if v, ok := saveB.(OutputVerifier); ok {
		if err := v.VerifyOutput(originalB, writtenB); err != nil {
			return nil, fmt.Errorf("Save B integrity check failed: %w", err)
		}
	}

	if err := verifyReloaded(writtenA, saveA.Generation(), refA, incomingToA); err != nil {
		return nil, fmt.Errorf("Save A output verification failed: %w", err)
	}
	if err := verifyReloaded(writtenB, saveB.Generation(), refB, incomingToB); err != nil {
		return nil, fmt.Errorf("Save B output verification failed: %w", err)
	}

	return &TradeResult{
		SaveA:      append([]byte(nil), writtenA...),
		SaveB:      append([]byte(nil), writtenB...),
		Evolutions: evolutions,
		Warnings:   preview.Warnings,
	}, nil
}

func verifyReloaded(data []byte, generation int, ref SlotRef, expected Pokemon) error {
	reloaded, err := LoadSaveFromBytes(data)
	if err != nil {
		return fmt.Errorf("rewritten save cannot be loaded: %w", err)
	}
	if reloaded.Generation() != generation {
		return fmt.Errorf("rewritten save was identified as Gen %d instead of Gen %d", reloaded.Generation(), generation)
	}
	store := storageFor(reloaded)
	got, err := store.ReadSlot(ref)
	if err != nil {
		return fmt.Errorf("rewritten %s cannot be read: %w", ref, err)
	}
	if got.GetSpecies() != expected.GetSpecies() {
		return fmt.Errorf("rewritten %s has species %d, expected %d", ref, got.GetSpecies(), expected.GetSpecies())
	}
	if !got.IsChecksumValid() {
		return fmt.Errorf("rewritten %s has an invalid checksum", ref)
	}
	return nil
}

// PreviewPartyTrade is the compatibility wrapper for the original party-only
// mobile and CLI paths. Indices are one-based.
func PreviewPartyTrade(saveA, saveB SaveFile, idxA, idxB int) *TradePreview {
	return PreviewTrade(saveA, saveB, Party(idxA), Party(idxB))
}

// ExecutePartyTrade is the compatibility wrapper for party-only callers.
func ExecutePartyTrade(saveA, saveB SaveFile, idxA, idxB int) (*TradeResult, error) {
	return ExecuteTrade(saveA, saveB, Party(idxA), Party(idxB))
}

package xgen

import (
	"fmt"

	"pktrade/internal/core"
	"pktrade/internal/gen1"
	"pktrade/internal/gen2"
	"pktrade/internal/gen3"
	"pktrade/internal/modern"
)

// Converter implements core.RecordConverter on top of this package.
type Converter struct{}

func init() { core.RegisterRecordConverter(Converter{}) }

// Transfer rebuilds p, a record of the `from` format, as a record of the `to`
// format sized for slot. The returned slice is the destination's loss list.
func (Converter) Transfer(p core.Pokemon, from, to core.CompatibilityKey, slot core.SlotKind) (core.Pokemon, []string, error) {
	out, losses, err := Convert(from, to, p, slot)
	if err != nil {
		return nil, nil, err
	}
	return out, losses.Strings(), nil
}

// Export decodes a record of the given format into the neutral form.
func Export(key core.CompatibilityKey, p core.Pokemon) (*Neutral, Losses, error) {
	switch key.Generation {
	case 1:
		gp, ok := p.(*gen1.Pokemon)
		if !ok {
			return nil, nil, wrongType(1, p)
		}
		return ExportGen1(gp)
	case 2:
		gp, ok := p.(*gen2.Pokemon)
		if !ok {
			return nil, nil, wrongType(2, p)
		}
		return ExportGen2(gp)
	case 3:
		gp, ok := p.(*gen3.Pokemon)
		if !ok {
			return nil, nil, wrongType(3, p)
		}
		return ExportGen3(gp)
	case 4, 5, 6, 7, 8:
		gp, ok := p.(*modern.Pokemon)
		if !ok {
			return nil, nil, wrongType(key.Generation, p)
		}
		return ExportModern(gp)
	}
	return nil, nil, fmt.Errorf("Gen %d is not supported by the conversion layer", key.Generation)
}

// Import builds a record in the destination format from the neutral form.
func Import(key core.CompatibilityKey, n *Neutral, slot core.SlotKind) (core.Pokemon, Losses, error) {
	xslot := Box
	if slot == core.PartySlot {
		xslot = Party
	}
	switch key.Generation {
	case 1:
		return ImportGen1(n, xslot)
	case 2:
		return ImportGen2(n, xslot)
	case 3:
		return ImportGen3(n, xslot)
	case 4, 5, 6, 7, 8:
		return ImportModern(n, modernFormat(key.Generation, key.Family), xslot)
	}
	return nil, nil, fmt.Errorf("Gen %d is not supported by the conversion layer", key.Generation)
}

// Convert translates a record from one format into another. The destination's
// loss list describes the source data the target format cannot represent.
func Convert(from, to core.CompatibilityKey, p core.Pokemon, slot core.SlotKind) (core.Pokemon, Losses, error) {
	neutral, _, err := Export(from, p)
	if err != nil {
		return nil, nil, err
	}
	out, losses, err := Import(to, neutral, slot)
	if err != nil {
		return nil, nil, err
	}
	return out, losses, nil
}

func modernFormat(gen int, family string) modern.Format {
	if gen == 4 && family == "hgss" {
		return modern.Format4HGSS
	}
	return modern.FormatFor(gen)
}

func wrongType(gen int, p core.Pokemon) error {
	return fmt.Errorf("a Gen %d target cannot read a %T record", gen, p)
}

// itemToCanonical maps a source item id to the canonical (Gen 4+) numbering.
func itemToCanonical(srcGen int, id uint16) (uint16, bool) {
	if id == 0 {
		return 0, true
	}
	return TranslateItem(srcGen, 4, id)
}

// itemForGen maps a canonical (Gen 4+) item id to a generation's numbering.
func itemForGen(gen int, canonical uint16) (uint16, bool) {
	if canonical == 0 {
		return 0, true
	}
	return TranslateItem(4, gen, canonical)
}

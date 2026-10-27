package core

// RecordConverter rebuilds a record of one format as a record of another.
// p belongs to `from`; the result belongs to `to`, sized for slot. The
// returned []string is the destination's loss list (empty means lossless).
// An error means the transfer must be refused and nothing written.
//
// The converter is registered by the package that owns the conversion tables
// (see internal/xgen), so a caller only needs to blank-import that package.
// It is the cross-generation analogue of CompatibilityProvider: instead of
// declaring which records are byte-identical, it declares how to translate
// them.
type RecordConverter interface {
	Transfer(p Pokemon, from, to CompatibilityKey, slot SlotKind) (Pokemon, []string, error)
}

var recordConverter RecordConverter

// RegisterRecordConverter installs the process-wide converter.
func RegisterRecordConverter(c RecordConverter) { recordConverter = c }

func converterAvailable() bool { return recordConverter != nil }

// convertRecord translates p from the `from` save into the `to` save's format.
func convertRecord(p Pokemon, from, to CompatibilityKey, slot SlotKind) (Pokemon, []string, error) {
	if recordConverter == nil {
		return nil, nil, errNoConverter(from, to)
	}
	return recordConverter.Transfer(p, from, to, slot)
}

func errNoConverter(from, to CompatibilityKey) error {
	return &convertError{from: from, to: to}
}

type convertError struct {
	from, to CompatibilityKey
}

func (e *convertError) Error() string {
	return "these saves cannot exchange records: cross-generation transfers are not available"
}

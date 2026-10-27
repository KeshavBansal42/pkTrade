package xgen

// Species availability per generation.
// A species is "available" in a generation if its National Dex number
// is within the generation's dex range AND the game supports it.

// MaxSpeciesID per generation (highest National Dex number the game stores).
var MaxSpeciesID = map[int]uint16{
	1: 151,
	2: 251,
	3: 386,
	4: 493,
	5: 649,
	6: 721,
	7: 807,
	8: 898,
}

// IsSpeciesAvailable reports whether a species exists in the given generation.
func IsSpeciesAvailable(gen int, nationalID uint16) bool {
	if nationalID == 0 {
		return false
	}
	maxID, ok := MaxSpeciesID[gen]
	if !ok {
		return false
	}
	return nationalID <= maxID
}

// IsFormAvailable checks whether a (species, form) pair exists in the given
// generation. For v1 we check that the form number is within a conservative
// range for the species; form 0 is always available if the species is.
//
// Form availability varies by species and generation (e.g. Rotom has forms
// in Gen 4+ but not Gen 1-3; Deoxys has different forms per game). A full
// table would need per-game form data; for now we use a simple heuristic:
// form 0 is always available; forms 1-7 are available for Gen 4+ if the
// species has alternate forms (which most don't).
func IsFormAvailable(gen int, nationalID uint16, form uint8) bool {
	if form == 0 {
		return IsSpeciesAvailable(gen, nationalID)
	}
	// Non-zero forms require at least Gen 4 (form field was introduced)
	if gen < 4 {
		return false
	}
	// Species that are known to have forms in Gen 4+ (conservative list)
	// This is a simplification; a full table would be per-game.
	hasForm := speciesWithForms[nationalID]
	if hasForm == 0 {
		return false
	}
	return form <= hasForm
}

// speciesWithForms maps National Dex number to the max form number
// available in Gen 4+. 0 means no alternate forms.
// This is a partial list covering common form-changing species.
var speciesWithForms = map[uint16]uint8{
	// Unown (forms A-Z)
	201: 27,
	// Deoxys
	386: 3,
	// Castform
	351: 1,
	// Cherrim
	421: 1,
	// Shellos / Gastrodon (regional)
	422: 1,
	423: 1,
	// Rotom (base + appliance forms)
	479: 6,
	// Giratina
	487: 1,
	// Shaymin
	492: 1,
	// Arceus
	493: 16,
	// Basculin
	550: 2,
	// Deerling / Sawsbuck
	585: 3,
	586: 3,
	// Darmanitan
	555: 1,
	// Tornadus / Thundurus / Landorus
	641: 1,
	642: 1,
	645: 1,
	// Kyurem
	646: 2,
	// Keldeo
	647: 1,
	// Meloetta
	648: 1,
	// Genesect
	649: 1,
	// Vivillon
	666: 18,
	// Furfrou
	676: 8,
	// Pumpkaboo / Gourgeist
	710: 3,
	711: 3,
	// Zygarde
	718: 2,
	// Oricorio
	741: 3,
	// Rockruff / Lycanroc
	744: 2,
	745: 2,
	// Wishiwashi
	746: 1,
	// Minior
	774: 1,
	// Mimikyu
	778: 1,
	// Necrozma
	800: 2,
	// Toxtricity
	849: 1,
	// Sinistea / Polteageist
	854: 1,
	855: 1,
	// Alcremie
	868: 9,
	// Urshifu
	892: 1,
	// Calyrex
	898: 2,
}

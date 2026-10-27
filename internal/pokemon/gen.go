package pokemon

// GrowthFor returns the growth-rate id of a species in a generation. Species
// outside the generation's tables fall back to the medium-fast group.
func GrowthFor(gen int, national uint16) byte {
	table := growthTable(gen)
	if table == nil || national == 0 || int(national) >= len(table) {
		return 3
	}
	return table[national]
}

func growthTable(gen int) []byte {
	switch gen {
	case 1:
		return Growth1[:]
	case 2:
		return Growth2[:]
	case 3:
		return Growth3[:]
	case 4:
		return Growth4[:]
	case 5:
		return Growth5[:]
	case 6:
		return Growth6[:]
	case 7:
		return Growth7[:]
	case 8:
		return Growth8[:]
	}
	return nil
}

// LevelFromExpFor converts experience to a level using the species' growth
// rate in the given generation.
func LevelFromExpFor(gen int, national uint16, exp uint32) byte {
	return LevelFromExp(GrowthFor(gen, national), exp)
}

// BaseStatsFor returns a species' base stats in a generation. A zero value is
// returned when the species does not exist in that generation.
func BaseStatsFor(gen int, national uint16) BaseStats {
	table := baseStatsTable(gen)
	if table == nil || national == 0 || int(national) >= len(table) {
		return BaseStats{}
	}
	return table[national]
}

func baseStatsTable(gen int) []BaseStats {
	switch gen {
	case 1:
		return BaseStats1[:]
	case 2:
		return BaseStats2[:]
	case 3:
		return BaseStats3[:]
	case 4:
		return BaseStats4[:]
	case 5:
		return BaseStats5[:]
	case 6:
		return BaseStats6[:]
	case 7:
		return BaseStats7[:]
	case 8:
		return BaseStats8[:]
	}
	return nil
}

package gen3

import (
	"encoding/binary"
)

type TradeEvo struct {
	TargetSpecies uint16
	RequiredItem  uint16
}

var evolutions = map[uint16][]TradeEvo{
	64:  {{65, 0}},       // Kadabra -> Alakazam
	67:  {{68, 0}},       // Machoke -> Machamp
	75:  {{76, 0}},       // Graveler -> Golem
	93:  {{94, 0}},       // Haunter -> Gengar
	61:  {{186, 228}},    // Poliwhirl + King's Rock -> Politoed
	79:  {{199, 228}},    // Slowpoke + King's Rock -> Slowking
	95:  {{208, 229}},    // Onix + Metal Coat -> Steelix
	123: {{212, 229}},    // Scyther + Metal Coat -> Scizor
	117: {{230, 223}},    // Seadra + Dragon Scale -> Kingdra
	137: {{233, 230}},    // Porygon + Up-Grade -> Porygon2
	391: {{392, 226}, {393, 227}}, // Clamperl + DeepSeaTooth -> Huntail / Scale -> Gorebyss
}

type BaseStats struct {
	HP, Atk, Def, Spe, SpA, SpD int
}

var evolvedStats = map[uint16]BaseStats{
	65:  {55, 50, 45, 120, 135, 95},
	68:  {90, 130, 80, 55, 65, 85},
	76:  {80, 120, 130, 45, 55, 65},
	94:  {60, 65, 60, 110, 130, 75},
	186: {90, 75, 75, 70, 90, 100},
	199: {95, 75, 80, 30, 100, 110},
	208: {75, 85, 200, 30, 55, 65},
	212: {70, 130, 100, 65, 55, 80},
	230: {75, 95, 95, 85, 95, 95},
	233: {85, 80, 90, 60, 105, 95},
	392: {55, 104, 105, 52, 94, 75},
	393: {55, 84, 105, 52, 114, 75},
}

var speciesNames = map[uint16]string{
	64: "KADABRA", 65: "ALAKAZAM",
	67: "MACHOKE", 68: "MACHAMP",
	75: "GRAVELER", 76: "GOLEM",
	93: "HAUNTER", 94: "GENGAR",
	61: "POLIWHIRL", 186: "POLITOED",
	79: "SLOWPOKE", 199: "SLOWKING",
	95: "ONIX", 208: "STEELIX",
	123: "SCYTHER", 212: "SCIZOR",
	117: "SEADRA", 230: "KINGDRA",
	137: "PORYGON", 233: "PORYGON2",
	391: "CLAMPERL", 392: "HUNTAIL", 393: "GOREBYSS",
}

func EncodeGen3String(s string, length int) []byte {
	res := make([]byte, length)
	for i := range res {
		res[i] = 0xFF
	}
	for i := 0; i < len(s) && i < length; i++ {
		c := s[i]
		if c == ' ' {
			res[i] = 0x00
		} else if c >= '0' && c <= '9' {
			res[i] = byte(0xA1 + (c - '0'))
		} else if c >= 'A' && c <= 'Z' {
			res[i] = byte(0xBB + (c - 'A'))
		} else if c >= 'a' && c <= 'z' {
			res[i] = byte(0xD5 + (c - 'a'))
		} else {
			res[i] = 0xAC
		}
	}
	return res
}

func ProcessTradeEvolution(data []byte) ([]byte, bool, error) {
	if len(data) < 100 {
		return data, false, nil // Only party pokemon can evolve since level/stats are needed
	}

	newData := make([]byte, len(data))
	copy(newData, data)

	pv := binary.LittleEndian.Uint32(newData[0x00:])
	otid := binary.LittleEndian.Uint32(newData[0x04:])
	key := pv ^ otid

	// 1. Decrypt
	decrypted := make([]byte, 48)
	for i := 0; i < 48; i += 4 {
		word := binary.LittleEndian.Uint32(newData[0x20+i : 0x24+i])
		binary.LittleEndian.PutUint32(decrypted[i:], word^key)
	}

	// 2. Unshuffle
	orderIdx := pv % 24
	order := subStructOrder[orderIdx]

	var blocks [4][]byte // G, A, E, M
	for i, c := range order {
		blockData := decrypted[i*12 : i*12+12]
		switch c {
		case 'G':
			blocks[0] = blockData
		case 'A':
			blocks[1] = blockData
		case 'E':
			blocks[2] = blockData
		case 'M':
			blocks[3] = blockData
		}
	}

	species := binary.LittleEndian.Uint16(blocks[0][0:2])
	heldItem := binary.LittleEndian.Uint16(blocks[0][2:4])

	evos, exists := evolutions[species]
	if !exists {
		return newData, false, nil
	}

	var evolvedInto uint16
	for _, evo := range evos {
		if evo.RequiredItem == 0 || evo.RequiredItem == heldItem {
			evolvedInto = evo.TargetSpecies
			if evo.RequiredItem != 0 {
				binary.LittleEndian.PutUint16(blocks[0][2:4], 0) // Consume item
			}
			break
		}
	}

	if evolvedInto == 0 {
		return newData, false, nil
	}

	// 3. Update species
	binary.LittleEndian.PutUint16(blocks[0][0:2], evolvedInto)

	// 4. Update nickname if it was the default
	currNick := decodeGen3String(newData[0x08:0x12])
	if currNick == speciesNames[species] {
		newNick := EncodeGen3String(speciesNames[evolvedInto], 10)
		copy(newData[0x08:0x12], newNick)
	}

	// 5. Recalculate stats
	level := newData[0x54]
	base := evolvedStats[evolvedInto]

	// IVs are in Misc (blocks[3]), bits 0-29.
	ivRaw := binary.LittleEndian.Uint32(blocks[3][4:8])
	hpIV := int(ivRaw & 0x1F)
	atkIV := int((ivRaw >> 5) & 0x1F)
	defIV := int((ivRaw >> 10) & 0x1F)
	speIV := int((ivRaw >> 15) & 0x1F)
	spaIV := int((ivRaw >> 20) & 0x1F)
	spdIV := int((ivRaw >> 25) & 0x1F)

	// EVs are in EVs (blocks[2])
	hpEV := int(blocks[2][0])
	atkEV := int(blocks[2][1])
	defEV := int(blocks[2][2])
	speEV := int(blocks[2][3])
	spaEV := int(blocks[2][4])
	spdEV := int(blocks[2][5])

	lvl := int(level)

	calcStat := func(base, iv, ev int, isHP bool) int {
		val := ((2*base + iv + (ev / 4)) * lvl) / 100
		if isHP {
			return val + lvl + 10
		}
		return val + 5
	}

	hpMax := calcStat(base.HP, hpIV, hpEV, true)
	atk := calcStat(base.Atk, atkIV, atkEV, false)
	def := calcStat(base.Def, defIV, defEV, false)
	spe := calcStat(base.Spe, speIV, speEV, false)
	spa := calcStat(base.SpA, spaIV, spaEV, false)
	spd := calcStat(base.SpD, spdIV, spdEV, false)

	// Nature multiplier (PV % 25)
	nature := pv % 25
	upStat := nature / 5
	downStat := nature % 5

	applyNature := func(statVal int, statIdx uint32) int {
		if upStat == downStat {
			return statVal
		}
		if upStat == statIdx {
			return (statVal * 110) / 100
		}
		if downStat == statIdx {
			return (statVal * 90) / 100
		}
		return statVal
	}

	// Indices: 0=Atk, 1=Def, 2=Spe, 3=SpA, 4=SpD
	atk = applyNature(atk, 0)
	def = applyNature(def, 1)
	spe = applyNature(spe, 2)
	spa = applyNature(spa, 3)
	spd = applyNature(spd, 4)

	// Preserve HP percentage
	oldMaxHP := binary.LittleEndian.Uint16(newData[0x58:])
	currHP := binary.LittleEndian.Uint16(newData[0x56:])
	var newCurrHP uint16
	if oldMaxHP > 0 {
		newCurrHP = uint16((int(currHP) * hpMax) / int(oldMaxHP))
	}
	if newCurrHP > uint16(hpMax) {
		newCurrHP = uint16(hpMax)
	}

	binary.LittleEndian.PutUint16(newData[0x56:], newCurrHP)
	binary.LittleEndian.PutUint16(newData[0x58:], uint16(hpMax))
	binary.LittleEndian.PutUint16(newData[0x5A:], uint16(atk))
	binary.LittleEndian.PutUint16(newData[0x5C:], uint16(def))
	binary.LittleEndian.PutUint16(newData[0x5E:], uint16(spe))
	binary.LittleEndian.PutUint16(newData[0x60:], uint16(spa))
	binary.LittleEndian.PutUint16(newData[0x62:], uint16(spd))

	// 6. Reshuffle and Checksum
	for i, c := range order {
		var blockData []byte
		switch c {
		case 'G':
			blockData = blocks[0]
		case 'A':
			blockData = blocks[1]
		case 'E':
			blockData = blocks[2]
		case 'M':
			blockData = blocks[3]
		}
		copy(decrypted[i*12:i*12+12], blockData)
	}

	var sum uint32
	for i := 0; i < 48; i += 2 {
		sum += uint32(binary.LittleEndian.Uint16(decrypted[i:]))
	}
	binary.LittleEndian.PutUint16(newData[0x1C:], uint16(sum))

	// 7. Re-encrypt
	for i := 0; i < 48; i += 4 {
		word := binary.LittleEndian.Uint32(decrypted[i : i+4])
		binary.LittleEndian.PutUint32(newData[0x20+i:], word^key)
	}

	return newData, true, nil
}

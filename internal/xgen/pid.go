package xgen

import (
	"math"

	"pktrade/internal/pokemon"
)

// dvFromIV keeps the four high bits of a 5-bit IV. A Gen 1/2 DV is 4 bits.
func dvFromIV(iv byte) byte { return (iv >> 1) & 0x0F }

// ivFromDV reproduces the official DV expansion: double and set the low bit,
// so DV 15 becomes the maximum IV 31 and DV 0 becomes IV 1 (Gen 1/2 has no
// IV 0).
func ivFromDV(dv byte) byte { return dv<<1 | 1 }

// hpDV is implied by the four stored nibbles; it is never stored on its own.
func hpDV(atk, def, spe, spc byte) byte {
	return (atk&1)<<3 | (def&1)<<2 | (spe&1)<<1 | spc&1
}

// statExpFromEV converts a Gen 3+ EV to the Gen 1/2 stat-experience that adds
// the same amount to the stat: both scale the contribution by sqrt()/4.
func statExpFromEV(ev uint16) uint16 {
	v := uint32(ev) * uint32(ev)
	if v > 65535 {
		return 65535
	}
	return uint16(v)
}

// evFromStatExp is the reverse of statExpFromEV.
func evFromStatExp(exp uint16) uint16 {
	v := uint16(math.Sqrt(float64(exp)))
	for uint32(v+1)*uint32(v+1) <= uint32(exp) {
		v++
	}
	return v
}

// genderFromLow returns the gender implied by a personality low byte (or a
// Gen 1/2 Attack DV) and a species gender ratio.
func genderFromLow(ratio, low byte) byte {
	switch ratio {
	case 255:
		return 2
	case 254:
		return 1
	case 0:
		return 0
	}
	if low < ratio {
		return 1
	}
	return 0
}

func genderRatio(national uint16) byte {
	if national == 0 || int(national) >= len(pokemon.GenderRatio) {
		return 255
	}
	return pokemon.GenderRatio[national]
}

// genderFromPID derives the gender of a Gen 3+ record.
func genderFromPID(national uint16, pid uint32) byte {
	return genderFromLow(genderRatio(national), byte(pid&0xFF))
}

// genderFromDV derives the gender of a Gen 1/2 record from its Attack DV.
func genderFromDV(national uint16, atkDV byte) byte {
	ratio := genderRatio(national)
	switch ratio {
	case 255:
		return 2
	case 254:
		return 1
	case 0:
		return 0
	}
	if int(atkDV)*2 < int(ratio>>3) {
		return 1
	}
	return 0
}

func genderOK(ratio, low, want byte) bool { return genderFromLow(ratio, low) == want }

// dvForGender nudges an Attack DV to the nearest value that produces the
// wanted gender. It reports whether the value had to change.
func dvForGender(national uint16, want, atkDV byte) (byte, bool) {
	if genderFromDV(national, atkDV) == want {
		return atkDV, false
	}
	for dv := int(atkDV) - 1; dv >= 0; dv-- {
		if genderFromDV(national, byte(dv)) == want {
			return byte(dv), true
		}
	}
	for dv := int(atkDV) + 1; dv <= 15; dv++ {
		if genderFromDV(national, byte(dv)) == want {
			return byte(dv), true
		}
	}
	return atkDV, false
}

// isShiny reports whether a Gen 3+ personality and trainer id form a shiny.
func isShiny(tid, sid uint16, pid uint32) bool {
	value := tid ^ sid ^ uint16(pid>>16) ^ uint16(pid)
	return value < 8
}

// sidForShiny returns the Secret ID that makes the personality shiny for the
// given trainer id.
func sidForShiny(tid uint16, pid uint32) uint16 {
	return tid ^ uint16(pid>>16) ^ uint16(pid)
}

// sidForNotShiny returns the Secret ID that makes the personality non-shiny.
func sidForNotShiny(tid uint16, pid uint32) uint16 {
	return sidForShiny(tid, pid) ^ 0xFFFF
}

// gen2Shiny reports whether the DVs match the Gen 2 shiny pattern.
func gen2Shiny(dvs uint16) bool {
	def := byte((dvs >> 4) & 0x0F)
	spe := byte((dvs >> 8) & 0x0F)
	spc := byte((dvs >> 12) & 0x0F)
	atk := byte(dvs & 0x0F)
	return def == 10 && spe == 10 && spc == 10 && (atk == 2 || atk == 3 || atk == 6 || atk == 7 ||
		atk == 10 || atk == 11 || atk == 14 || atk == 15)
}

// buildPID3 finds a personality value that yields the wanted nature and
// gender (and shininess when hasSID is true). When the source has no Secret
// ID the caller invents one from the shiny flag, so shininess is always exact.
func buildPID3(national uint16, tid, sid uint16, nature, gender byte, shiny, hasSID bool) (uint32, bool) {
	ratio := genderRatio(national)
	if !hasSID {
		for b := 0; b < 65536; b++ {
			if !genderOK(ratio, byte(b), gender) {
				continue
			}
			a := smallestA(int(nature), b)
			return uint32(a)<<16 | uint32(b), true
		}
		return 0, false
	}
	for b := 0; b < 65536; b++ {
		if !genderOK(ratio, byte(b), gender) {
			continue
		}
		base := uint16(tid) ^ sid ^ uint16(b)
		for target := 0; target < 256; target++ {
			if (target < 8) != shiny {
				continue
			}
			a := base ^ uint16(target)
			pid := uint32(a)<<16 | uint32(b)
			if pid%25 == uint32(nature) {
				return pid, true
			}
		}
	}
	return 0, false
}

// smallestA returns the smallest high half whose 32-bit value, combined with
// the low half b, has the wanted nature (value % 25). 65536 mod 25 is 11 and
// 11 is invertible mod 25, so a unique residue always exists.
func smallestA(nature, b int) int {
	// a*65536 + b ≡ nature (mod 25)  =>  a*11 ≡ nature-b
	residue := ((nature-b)%25 + 25) % 25
	return (residue * 16) % 25 // 16 is the inverse of 11 modulo 25
}

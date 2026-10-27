package xgen

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/gen1"
	"pktrade/internal/pokemon"
)

// maxSpeciesGen1 is the highest National Dex id Generation I can store.
const maxSpeciesGen1 = 151

// ExportGen1 decodes a Generation I record.
func ExportGen1(p *gen1.Pokemon) (*Neutral, Losses, error) {
	var losses Losses
	national := p.NationalID()
	if national == 0 {
		return nil, losses, ErrNoSpecies
	}
	n := &Neutral{SourceGen: 1, SourceParty: !p.Stored}
	n.Species = national
	n.Nick = p.NicknameStr
	n.OT = p.OTNameStr
	n.TID = p.OTID
	n.Exp = p.Exp
	n.Level = p.LevelVal

	dv := p.DVs
	atk := byte(dv & 0x0F)
	def := byte((dv >> 4) & 0x0F)
	spe := byte((dv >> 8) & 0x0F)
	spc := byte((dv >> 12) & 0x0F)
	n.IVs = [6]byte{
		ivFromDV(hpDV(atk, def, spe, spc)),
		ivFromDV(atk), ivFromDV(def), ivFromDV(spe), ivFromDV(spc), ivFromDV(spc),
	}
	statExp := func(off int) uint16 { return binary.BigEndian.Uint16(p.RawData[off:]) }
	n.EVs = [6]uint16{
		evFromStatExp(statExp(0x11)), evFromStatExp(statExp(0x13)),
		evFromStatExp(statExp(0x15)), evFromStatExp(statExp(0x17)),
		evFromStatExp(statExp(0x19)), evFromStatExp(statExp(0x19)),
	}
	n.Gender = genderFromDV(national, atk)
	n.Shiny = gen2Shiny(dv)
	for i := 0; i < 4; i++ {
		n.Moves[i] = uint16(p.MoveList[i])
		n.MovePP[i] = p.PPMoves[i]
	}
	losses.add("stats", "Generation I stored 4-bit DVs and stat experience, not 5-bit IVs and EVs; the low bit of each stat could not be carried")
	losses.add("nature, ability, held item, capture ball and met data", "these fields do not exist in Generation I")
	return n, losses, nil
}

// ImportGen1 builds a Generation I record from the neutral form.
func ImportGen1(n *Neutral, slot Slot) (*gen1.Pokemon, Losses, error) {
	var losses Losses
	if n.Species == 0 {
		return nil, losses, ErrNoSpecies
	}
	if n.IsEgg {
		return nil, losses, ErrEgg
	}
	if n.Species > maxSpeciesGen1 {
		return nil, losses, ErrSpeciesAbsent
	}
	internal := pokemon.Internal1(n.Species)
	if internal == 0 {
		return nil, losses, ErrSpeciesAbsent
	}

	body := make([]byte, gen1.BoxRecordSize)
	body[0x00] = byte(internal)
	body[0x03] = n.Level
	types := pokemon.TypeGen1[n.Species]
	body[0x05] = types[0]
	body[0x06] = types[1]
	body[0x07] = pokemon.CatchRateGen1[n.Species]

	for i := 0; i < 4; i++ {
		move := n.Moves[i]
		if move != 0 && move > 165 {
			losses.add(fmt.Sprintf("move %d", i+1), "the move does not exist in Generation I")
			move = 0
		}
		body[0x08+i] = byte(move)
	}
	if n.MovePPUp != [4]byte{} {
		losses.add("PP-Ups", "Generation I stores no PP-Up count")
	}

	binary.BigEndian.PutUint16(body[0x0C:], n.TID)
	body[0x0E] = byte(n.Exp >> 16)
	body[0x0F] = byte(n.Exp >> 8)
	body[0x10] = byte(n.Exp)

	binary.BigEndian.PutUint16(body[0x11:], statExpFromEV(n.EVs[0]))
	binary.BigEndian.PutUint16(body[0x13:], statExpFromEV(n.EVs[1]))
	binary.BigEndian.PutUint16(body[0x15:], statExpFromEV(n.EVs[2]))
	binary.BigEndian.PutUint16(body[0x17:], statExpFromEV(n.EVs[3]))
	spc := n.EVs[4]
	if n.EVs[5] < spc {
		spc = n.EVs[5]
	}
	binary.BigEndian.PutUint16(body[0x19:], statExpFromEV(spc))

	atk := dvFromIV(n.IVs[1])
	def := dvFromIV(n.IVs[2])
	spe := dvFromIV(n.IVs[3])
	spcDV := dvFromIV(n.IVs[4])
	if n.IVs[5] < n.IVs[4] {
		spcDV = dvFromIV(n.IVs[5])
	}
	if n.IVs[4] != n.IVs[5] {
		losses.add("special stats", "Generation I stores a single Special stat; Sp. Atk and Sp. Def cannot differ")
	}
	if adjusted, changed := dvForGender(n.Species, n.Gender, atk); changed {
		atk = adjusted
		losses.add("gender", "the gender was encoded into the Attack DV")
	}
	dvs := uint16(atk) | uint16(def)<<4 | uint16(spe)<<8 | uint16(spcDV)<<12
	binary.BigEndian.PutUint16(body[0x1B:], dvs)
	copy(body[0x1D:0x21], n.MovePP[:])

	nick := gen1.EncodeString(n.Nick, gen1.NameLength)
	ot := gen1.EncodeString(n.OT, gen1.NameLength)
	p, err := gen1.ParsePokemon(body, nick, ot)
	if err != nil {
		return nil, losses, err
	}
	party, err := p.ToPartyData()
	if err != nil {
		return nil, losses, err
	}
	maxHP := binary.BigEndian.Uint16(party[0x22:])
	binary.BigEndian.PutUint16(body[0x01:], maxHP)
	p, err = gen1.ParsePokemon(body, nick, ot)
	if err != nil {
		return nil, losses, err
	}
	if slot == Party {
		binary.BigEndian.PutUint16(party[0x01:], maxHP)
		p, err = gen1.ParsePokemon(party, nick, ot)
		if err != nil {
			return nil, losses, err
		}
	}
	reportAbsent(&losses, "Generation I", n, "held item", "capture ball", "nature", "ability", "friendship", "Pokérus", "met data", "Secret ID")
	return p, losses, nil
}

// reportAbsent records the fields a destination generation cannot store.
func reportAbsent(losses *Losses, gen string, n *Neutral, fields ...string) {
	for _, field := range fields {
		losses.add(field, "not representable in "+gen)
	}
}

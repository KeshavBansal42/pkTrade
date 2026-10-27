package xgen

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/gen2"
)

// maxSpeciesGen2 is the highest National Dex id Generation II can store.
const maxSpeciesGen2 = 251

// ExportGen2 decodes a Generation II record.
func ExportGen2(p *gen2.Pokemon) (*Neutral, Losses, error) {
	var losses Losses
	national := p.NationalID()
	if national == 0 {
		return nil, losses, ErrNoSpecies
	}
	n := &Neutral{SourceGen: 2, SourceParty: !p.Stored}
	n.Species = national
	n.Nick = p.NicknameStr
	n.OT = p.OTNameStr
	n.TID = p.OTID
	n.Exp = p.Exp
	n.Level = p.LevelVal
	n.IsEgg = p.IsEgg
	n.Friendship = p.Friendship
	n.HasFriendship = true
	n.Pokerus = p.Pokerus
	n.HasPokerus = true

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
		evFromStatExp(statExp(0x0B)), evFromStatExp(statExp(0x0D)),
		evFromStatExp(statExp(0x0F)), evFromStatExp(statExp(0x11)),
		evFromStatExp(statExp(0x13)), evFromStatExp(statExp(0x13)),
	}
	n.Gender = genderFromDV(national, atk)
	n.Shiny = gen2Shiny(dv)
	if item, ok := itemToCanonical(2, uint16(p.HeldItem)); ok {
		n.HeldItem = item
	} else {
		losses.add("held item", "the item has no modern equivalent")
	}
	for i := 0; i < 4; i++ {
		n.Moves[i] = uint16(p.Moves[i])
		n.MovePP[i] = p.PPMoves[i]
	}
	losses.add("stats", "Generation II stored 4-bit DVs and stat experience, not 5-bit IVs and EVs; the low bit of each stat could not be carried")
	losses.add("nature, ability, capture ball and met data", "these fields do not exist in Generation II")
	return n, losses, nil
}

// ImportGen2 builds a Generation II record from the neutral form.
func ImportGen2(n *Neutral, slot Slot) (*gen2.Pokemon, Losses, error) {
	var losses Losses
	if n.Species == 0 {
		return nil, losses, ErrNoSpecies
	}
	if n.IsEgg {
		return nil, losses, ErrEgg
	}
	if n.Species > maxSpeciesGen2 {
		return nil, losses, ErrSpeciesAbsent
	}

	body := make([]byte, gen2.BoxRecordSize)
	body[0x00] = byte(n.Species)
	if n.HeldItem != 0 {
		if item, ok := itemForGen(2, n.HeldItem); ok {
			body[0x01] = byte(item)
		} else {
			losses.add("held item", "the item does not exist in Generation II")
		}
	}
	pp := n.MovePP
	for i := 0; i < 4; i++ {
		move := n.Moves[i]
		if move != 0 && move > 251 {
			losses.add(fmt.Sprintf("move %d", i+1), "the move does not exist in Generation II")
			move = 0
			pp[i] = 0
		}
		body[0x02+i] = byte(move)
	}
	if n.MovePPUp != [4]byte{} {
		losses.add("PP-Ups", "Generation II stores no PP-Up count")
	}
	binary.BigEndian.PutUint16(body[0x06:], n.TID)
	body[0x08] = byte(n.Exp >> 16)
	body[0x09] = byte(n.Exp >> 8)
	body[0x0A] = byte(n.Exp)

	binary.BigEndian.PutUint16(body[0x0B:], statExpFromEV(n.EVs[0]))
	binary.BigEndian.PutUint16(body[0x0D:], statExpFromEV(n.EVs[1]))
	binary.BigEndian.PutUint16(body[0x0F:], statExpFromEV(n.EVs[2]))
	binary.BigEndian.PutUint16(body[0x11:], statExpFromEV(n.EVs[3]))
	spc := n.EVs[4]
	if n.EVs[5] < spc {
		spc = n.EVs[5]
	}
	binary.BigEndian.PutUint16(body[0x13:], statExpFromEV(spc))

	atk := dvFromIV(n.IVs[1])
	def := dvFromIV(n.IVs[2])
	spe := dvFromIV(n.IVs[3])
	spcDV := dvFromIV(n.IVs[4])
	if n.IVs[5] < n.IVs[4] {
		spcDV = dvFromIV(n.IVs[5])
	}
	if n.IVs[4] != n.IVs[5] {
		losses.add("special stats", "Generation II stores a single Special stat; Sp. Atk and Sp. Def cannot differ")
	}
	if adjusted, changed := dvForGender(n.Species, n.Gender, atk); changed {
		atk = adjusted
		losses.add("gender", "the gender was encoded into the Attack DV")
	}
	dvs := uint16(atk) | uint16(def)<<4 | uint16(spe)<<8 | uint16(spcDV)<<12
	if n.Shiny {
		shinyAtk, genderKept := gen2ShinyAtkFor(n.Species, n.Gender, atk)
		dvs = uint16(shinyAtk) | 10<<4 | 10<<8 | 10<<12
		losses.add("shiny", "the shiny DV pattern replaced the source IVs to keep the Pokémon shiny")
		if !genderKept {
			losses.add("gender", "the shiny DV pattern could not also keep the gender")
		}
	} else if gen2Shiny(dvs) {
		if safe, changed := nonShinyAtk(atk); changed {
			dvs = uint16(safe) | uint16(def)<<4 | uint16(spe)<<8 | uint16(spcDV)<<12
			losses.add("shiny", "the Attack DV was nudged so a non-shiny Pokémon does not turn shiny")
		}
	}
	binary.BigEndian.PutUint16(body[0x15:], dvs)
	copy(body[0x17:0x1B], pp[:])
	if n.HasFriendship {
		body[0x1B] = n.Friendship
	} else {
		body[0x1B] = 70
	}
	if n.HasPokerus {
		body[0x1C] = n.Pokerus
	}
	body[0x1F] = n.Level

	nick := gen2.EncodeString(n.Nick, gen2.NameLength)
	ot := gen2.EncodeString(n.OT, gen2.NameLength)
	p, err := gen2.ParsePokemonMarker(body, nick, ot, 0)
	if err != nil {
		return nil, losses, err
	}
	if slot == Party {
		party, err := p.ToPartyData()
		if err != nil {
			return nil, losses, err
		}
		p, err = gen2.ParsePokemonMarker(party, nick, ot, 0)
		if err != nil {
			return nil, losses, err
		}
	}
	reportAbsent(&losses, "Generation II", n, "capture ball", "nature", "ability", "met data", "Secret ID")
	return p, losses, nil
}

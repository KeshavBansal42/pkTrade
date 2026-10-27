package xgen

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/modern"
)

// ExportModern decodes a Generation 4-8 record.
func ExportModern(p *modern.Pokemon) (*Neutral, Losses, error) {
	var losses Losses
	gen := p.Format.Generation
	n := &Neutral{SourceGen: gen, SourceParty: !p.Stored}
	national := p.NationalID()
	if national == 0 {
		return nil, losses, ErrNoSpecies
	}
	n.Species = national
	n.Nick = p.NicknameStr
	n.OT = p.OTNameStr
	n.TID = uint16(p.OTID & 0xFFFF)
	n.SID = uint16(p.OTID >> 16)
	n.HasSID = true
	n.Exp = p.Exp
	n.Level = p.LevelVal
	n.PID = p.PID
	n.HasPID = true
	n.Nature = p.NatureVal
	n.HasNature = true

	word := binary.LittleEndian.Uint32(p.Data[p.Format.IV:])
	n.IVs = [6]byte{
		byte(word & 0x1F), byte((word >> 5) & 0x1F), byte((word >> 10) & 0x1F),
		byte((word >> 15) & 0x1F), byte((word >> 20) & 0x1F), byte((word >> 25) & 0x1F),
	}
	n.AbilitySlot = byte((word >> 31) & 1)
	n.IsEgg = p.IsEgg
	for i := 0; i < 6; i++ {
		n.EVs[i] = uint16(p.Data[p.Format.EV[i]])
	}
	n.Gender = genderFromPID(national, p.PID)
	n.Shiny = isShiny(n.TID, n.SID, p.PID)
	n.Moves = p.GetMoves()
	n.MovePP = p.GetPP()
	n.MovePPUp = p.GetPPUp()
	// The Gen 4+ item numbering is the canonical one, so no translation is
	// needed here.
	n.HeldItem = p.HeldItem
	if ball, ok := p.GetCaptureBall(); ok {
		n.Ball = ball
		n.HasBall = true
	}
	if n.IsEgg {
		return nil, losses, ErrEgg
	}
	return n, losses, nil
}

// ImportModern builds a Generation 4-8 record from the neutral form.
func ImportModern(n *Neutral, format modern.Format, slot Slot) (*modern.Pokemon, Losses, error) {
	var losses Losses
	gen := format.Generation
	if n.Species == 0 {
		return nil, losses, ErrNoSpecies
	}
	if n.IsEgg {
		return nil, losses, ErrEgg
	}
	if limit, ok := MaxSpeciesID[gen]; ok && n.Species > limit {
		return nil, losses, ErrSpeciesAbsent
	}

	nature := byte(0)
	if n.HasNature {
		nature = n.Nature
	} else if n.HasPID {
		nature = byte(n.PID % 25)
	} else {
		losses.add("nature", "the source has no nature; a Hardy nature was chosen")
	}

	derives := format.Nature < 0 // Gen 4 derives the nature from the personality value
	pid := uint32(0)
	rebuilt := false
	if n.HasPID && (!derives || n.PID%25 == uint32(nature)) {
		pid = n.PID
	} else {
		rebuilt = true
		built, ok := buildPID3(n.Species, n.TID, n.SID, nature, n.Gender, n.Shiny, n.HasSID)
		if !ok {
			built, ok = buildPID3(n.Species, n.TID, n.SID, nature, n.Gender, false, n.HasSID)
			losses.add("shiny", "no personality value could keep the Pokémon shiny and match its nature; the shiny was not kept")
		}
		if !ok {
			built = n.PID
			if built == 0 {
				built = 1
			}
			losses.add("nature and gender", "no personality value matched; the nature or gender may differ")
		}
		pid = built
		if n.HasPID {
			losses.add("personality", "the personality value was rebuilt to keep the nature; PID-derived forms (Unown, Spinda, etc.) may change")
		}
	}
	abilitySlot := n.AbilitySlot
	if rebuilt {
		abilitySlot = byte(pid & 1)
	}
	sid := n.SID
	if !n.HasSID {
		if n.Shiny {
			sid = sidForShiny(n.TID, pid)
		} else {
			sid = sidForNotShiny(n.TID, pid)
		}
		losses.add("Secret ID", "the source has no Secret ID; one was created")
	}

	p := modern.New(format, slot == Party)
	if err := p.SetSpecies(n.Species); err != nil {
		return nil, losses, err
	}
	p.SetPID(pid)
	p.SetOTID(uint32(n.TID) | uint32(sid)<<16)
	p.SetExp(n.Exp)
	p.SetNature(nature)
	p.SetIVs(n.IVs)
	p.SetIVFlags(false, abilitySlot)
	evs, evChanged := capEVs(n.EVs, 252, 510)
	if evChanged {
		losses.add("EVs", "the EVs were scaled or clamped to Generation 4+'s 252 per-stat / 510 total cap")
	}
	p.SetEVs(evs)
	moves, pp, ppup := n.Moves, n.MovePP, n.MovePPUp
	for i := 0; i < 4; i++ {
		if moves[i] != 0 && moves[i] > MaxMoveID[gen] {
			losses.add(fmt.Sprintf("move %d", i+1), fmt.Sprintf("the move does not exist in Generation %d", gen))
			moves[i] = 0
			pp[i] = 0
			ppup[i] = 0
		}
	}
	p.SetMoves(moves)
	p.SetPP(pp)
	p.SetPPUp(ppup)
	p.SetNickname(n.Nick)
	p.SetOTName(n.OT)

	if n.HeldItem != 0 {
		if max, ok := MaxItemID[gen]; ok && n.HeldItem > max {
			losses.add("held item", fmt.Sprintf("the item does not exist in Generation %d", gen))
		} else {
			p.SetItem(n.HeldItem)
		}
	}
	if n.HasBall && n.Ball > 0 {
		p.SetBall(n.Ball)
	}

	p.Refresh()
	if slot == Party {
		if err := p.RecomputeStats(); err != nil {
			return nil, losses, err
		}
	}
	size := format.StoredSize
	if slot == Party {
		size = format.PartySize
	}
	raw, err := p.Encode(size)
	if err != nil {
		return nil, losses, err
	}
	out, err := modern.Parse(format, raw)
	if err != nil {
		return nil, losses, err
	}
	losses.add("friendship, Pokérus, met data and language", "not carried across generations by this converter")
	return out, losses, nil
}

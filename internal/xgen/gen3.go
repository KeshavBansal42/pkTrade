package xgen

import (
	"fmt"

	"pktrade/internal/gen3"
	"pktrade/internal/pokemon"
)

// maxSpeciesGen3 is the highest National Dex id Generation III can store.
const maxSpeciesGen3 = 386

// ExportGen3 decodes a Generation III record.
func ExportGen3(p *gen3.Pokemon) (*Neutral, Losses, error) {
	var losses Losses
	if p.National == 0 {
		return nil, losses, ErrNoSpecies
	}
	n := &Neutral{SourceGen: 3, SourceParty: len(p.RawData) == 100}
	n.Species = p.National
	n.Nick = p.Nickname
	n.OT = p.OTName
	n.TID = uint16(p.OTID & 0xFFFF)
	n.SID = uint16(p.OTID >> 16)
	n.HasSID = true
	n.Exp = p.Experience
	n.Level = p.Level
	n.PID = p.PersonalityValue
	n.HasPID = true
	n.Nature = p.Nature
	n.HasNature = true
	n.IVs = p.IVs
	for i := 0; i < 6; i++ {
		n.EVs[i] = uint16(p.EVs[i])
	}
	n.AbilitySlot = p.AbilitySlot
	n.Gender = genderFromPID(n.Species, p.PersonalityValue)
	n.Shiny = isShiny(n.TID, n.SID, p.PersonalityValue)
	for i := 0; i < 4; i++ {
		n.Moves[i] = p.Moves[i]
		n.MovePP[i] = p.PP[i]
		n.MovePPUp[i] = (p.PPUps >> (2 * i)) & 0x03
	}
	if item, ok := itemToCanonical(3, p.HeldItem); ok {
		n.HeldItem = item
	} else if p.HeldItem != 0 {
		losses.add("held item", "the item has no modern equivalent")
	}
	if p.BallKnown {
		n.Ball = p.BallID
		n.HasBall = true
	}
	n.Friendship = p.Friendship
	n.HasFriendship = true
	n.Pokerus = p.Pokerus
	n.HasPokerus = true
	n.MetLevel = p.MetLevel
	n.MetLocation = p.MetLocation
	n.HasMet = true
	n.IsEgg = p.IsEgg
	if n.IsEgg {
		return nil, losses, ErrEgg
	}
	return n, losses, nil
}

// ImportGen3 builds a Generation III record from the neutral form.
func ImportGen3(n *Neutral, slot Slot) (*gen3.Pokemon, Losses, error) {
	var losses Losses
	if n.Species == 0 {
		return nil, losses, ErrNoSpecies
	}
	if n.IsEgg {
		return nil, losses, ErrEgg
	}
	if n.Species > maxSpeciesGen3 {
		return nil, losses, ErrSpeciesAbsent
	}
	internal := pokemon.Internal3(n.Species)
	if internal == 0 {
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

	pid := uint32(0)
	if n.HasPID && n.PID%25 == uint32(nature) {
		pid = n.PID
	} else {
		built, ok := buildPID3(n.Species, n.TID, n.SID, nature, n.Gender, n.Shiny, n.HasSID)
		if !ok {
			built, _ = buildPID3(n.Species, n.TID, n.SID, nature, n.Gender, false, n.HasSID)
			losses.add("shiny", "no personality value could keep the Pokémon shiny and match its nature; the shiny was not kept")
		}
		pid = built
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

	p := &gen3.Pokemon{
		PersonalityValue: pid,
		OTID:             uint32(n.TID) | uint32(sid)<<16,
		Nickname:         n.Nick,
		OTName:           n.OT,
		Language:         2,
		Species:          internal,
		National:         n.Species,
		Experience:       n.Exp,
		Nature:           nature,
		IVs:              n.IVs,
		Friendship:       n.Friendship,
		Pokerus:          n.Pokerus,
		MetLocation:      n.MetLocation,
		MetLevel:         n.MetLevel,
		AbilitySlot:      n.AbilitySlot,
	}
	if !n.HasFriendship {
		p.Friendship = 70
	}
	p.Level = pokemon.LevelFromExpFor(3, n.Species, n.Exp)
	if p.Level < 1 {
		p.Level = 1
	}

	if n.HeldItem != 0 {
		if item, ok := itemForGen(3, n.HeldItem); ok {
			p.HeldItem = item
		} else {
			losses.add("held item", "the item does not exist in Generation III")
		}
	}
	for i := 0; i < 4; i++ {
		move := n.Moves[i]
		if move != 0 && move > 354 {
			losses.add(fmt.Sprintf("move %d", i+1), "the move does not exist in Generation III")
			move = 0
		}
		p.Moves[i] = move
		p.PP[i] = n.MovePP[i]
		p.PPUps |= (n.MovePPUp[i] & 0x03) << (2 * i)
	}
	for i := 0; i < 6; i++ {
		ev := n.EVs[i]
		if ev > 255 {
			ev = 255
			losses.add("EVs", "the EV total was above the Generation III cap and was clamped")
		}
		p.EVs[i] = byte(ev)
	}
	if n.HasBall && n.Ball >= 1 && n.Ball <= 12 {
		p.BallID = n.Ball
		p.BallKnown = true
	} else if n.HasBall {
		losses.add("capture ball", "the ball does not exist in Generation III; the record has no ball")
	}

	size := 80
	if slot == Party {
		size = 100
	}
	raw, err := p.Encode(size)
	if err != nil {
		return nil, losses, err
	}
	out, err := gen3.ParsePokemon(raw)
	if err != nil {
		return nil, losses, err
	}
	return out, losses, nil
}

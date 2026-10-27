package gen3

import (
	"encoding/binary"
	"fmt"

	"pktrade/internal/pokemon"
)

type Pokemon struct {
	RawData          []byte
	PersonalityValue uint32
	OTID             uint32
	BallID           byte
	BallKnown        bool
	Nickname         string
	Language         byte
	Flags            byte
	OTName           string
	Markings         byte
	DataChecksum     uint16

	Species    uint16 // internal Gen 3 species ID, as stored in the save
	National   uint16 // National Dex ID, 0 when the internal ID has no national mapping
	HeldItem   uint16
	Experience uint32

	Moves [4]uint16
	PP    [4]byte

	Level byte

	IVs    [6]byte // HP, Atk, Def, Spe, SpA, SpD
	EVs    [6]byte // HP, Atk, Def, Spe, SpA, SpD
	Nature byte

	PPUps       byte // packed PP-Up bonuses, two bits per move slot
	Friendship  byte
	Pokerus     byte
	MetLocation byte
	MetLevel    byte
	OriginGame  byte
	OTGender    byte
	AbilitySlot byte
	IsEgg       bool

	ChecksumValid bool
	BoxNum        int
	BoxSlot       int
}

var subStructOrder = [24]string{
	"GAEM", "GAME", "GEAM", "GEMA", "GMAE", "GMEA",
	"AGEM", "AGME", "AEGM", "AEMG", "AMGE", "AMEG",
	"EGAM", "EGMA", "EAGM", "EAMG", "EMGA", "EMAG",
	"MGAE", "MGEA", "MAGE", "MAEG", "MEGA", "MEAG",
}

type dataBlocks struct {
	decrypted []byte
	growth    []byte
	attacks   []byte
	evs       []byte
	misc      []byte
	order     [4]byte
	key       uint32
}

func ParsePokemon(data []byte) (*Pokemon, error) {
	if len(data) != 80 && len(data) != 100 {
		return nil, fmt.Errorf("gen 3 Pokémon record must be 80 or 100 bytes, got %d", len(data))
	}

	p := &Pokemon{RawData: append([]byte(nil), data...)}
	p.PersonalityValue = binary.LittleEndian.Uint32(data[0x00:])
	p.OTID = binary.LittleEndian.Uint32(data[0x04:])
	p.Nickname = decodeGen3String(data[0x08:0x12])
	p.Language = data[0x12]
	p.Flags = data[0x13]
	p.OTName = decodeGen3String(data[0x14:0x1B])
	p.Markings = data[0x1B]
	p.DataChecksum = binary.LittleEndian.Uint16(data[0x1C:])
	p.Nature = byte(p.PersonalityValue % 25)

	blocks, err := p.blocks()
	if err != nil {
		return nil, err
	}
	p.ChecksumValid = p.calculateChecksum(blocks.decrypted) == p.DataChecksum
	p.Species = binary.LittleEndian.Uint16(blocks.growth[0:2])
	p.National = pokemon.National3(p.Species)
	p.HeldItem = binary.LittleEndian.Uint16(blocks.growth[2:4])
	p.Experience = binary.LittleEndian.Uint32(blocks.growth[4:8])

	copy(p.Moves[:], []uint16{
		binary.LittleEndian.Uint16(blocks.attacks[0:2]),
		binary.LittleEndian.Uint16(blocks.attacks[2:4]),
		binary.LittleEndian.Uint16(blocks.attacks[4:6]),
		binary.LittleEndian.Uint16(blocks.attacks[6:8]),
	})
	copy(p.PP[:], blocks.attacks[8:12])

	if p.National != 0 && p.National < uint16(len(pokemon.Growth3)) {
		p.Level = pokemon.LevelFromExp(pokemon.Growth3[p.National], p.Experience)
	}
	if len(data) == 100 {
		p.Level = data[0x54]
	}

	ivRaw := binary.LittleEndian.Uint32(blocks.misc[4:8])
	p.IVs = [6]byte{
		byte(ivRaw & 0x1F),
		byte((ivRaw >> 5) & 0x1F),
		byte((ivRaw >> 10) & 0x1F),
		byte((ivRaw >> 15) & 0x1F),
		byte((ivRaw >> 20) & 0x1F),
		byte((ivRaw >> 25) & 0x1F),
	}
	p.EVs = [6]byte{blocks.evs[0], blocks.evs[1], blocks.evs[2], blocks.evs[3], blocks.evs[4], blocks.evs[5]}

	p.PPUps = blocks.growth[8]
	p.Friendship = blocks.growth[9]
	p.Pokerus = blocks.misc[0]
	p.MetLocation = blocks.misc[1]
	p.AbilitySlot = byte((ivRaw >> 31) & 1)
	p.IsEgg = (ivRaw>>30)&1 == 1

	if p.ChecksumValid {
		originsInfo := binary.LittleEndian.Uint16(blocks.misc[2:4])
		p.MetLevel = byte(originsInfo & 0x7F)
		p.OTGender = byte((originsInfo >> 7) & 1)
		p.OriginGame = byte((originsInfo >> 8) & 0x0F)
		ballID := byte((originsInfo >> 11) & 0x0F)
		if ballID >= 1 && ballID <= 12 {
			p.BallID = ballID
			p.BallKnown = true
		}
	}

	return p, nil
}

func (p *Pokemon) blocks() (*dataBlocks, error) {
	if len(p.RawData) < 80 {
		return nil, fmt.Errorf("gen 3 record is truncated")
	}
	key := p.PersonalityValue ^ p.OTID
	decrypted := make([]byte, 48)
	for i := 0; i < 48; i += 4 {
		word := binary.LittleEndian.Uint32(p.RawData[0x20+i : 0x24+i])
		binary.LittleEndian.PutUint32(decrypted[i:], word^key)
	}

	order := subStructOrder[p.PersonalityValue%24]
	b := &dataBlocks{decrypted: decrypted, key: key, order: [4]byte{order[0], order[1], order[2], order[3]}}
	for i, block := range order {
		blockData := decrypted[i*12 : i*12+12]
		switch block {
		case 'G':
			b.growth = blockData
		case 'A':
			b.attacks = blockData
		case 'E':
			b.evs = blockData
		case 'M':
			b.misc = blockData
		}
	}
	if b.growth == nil || b.attacks == nil || b.evs == nil || b.misc == nil {
		return nil, fmt.Errorf("gen 3 record has an invalid substructure order")
	}
	return b, nil
}

func (p *Pokemon) calculateChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+2 <= len(data); i += 2 {
		sum += uint32(binary.LittleEndian.Uint16(data[i:]))
	}
	return uint16(sum)
}

// writeBlocks reshuffles b.decrypted into p.RawData, recalculates the record
// checksum, and re-encrypts the 48-byte data block.
func (p *Pokemon) writeBlocks(b *dataBlocks) {
	for i, block := range b.order {
		var source []byte
		switch block {
		case 'G':
			source = b.growth
		case 'A':
			source = b.attacks
		case 'E':
			source = b.evs
		case 'M':
			source = b.misc
		}
		copy(b.decrypted[i*12:i*12+12], source)
	}
	p.DataChecksum = p.calculateChecksum(b.decrypted)
	binary.LittleEndian.PutUint16(p.RawData[0x1C:], p.DataChecksum)
	for i := 0; i < 48; i += 4 {
		word := binary.LittleEndian.Uint32(b.decrypted[i:])
		binary.LittleEndian.PutUint32(p.RawData[0x20+i:], word^b.key)
	}
}

// ToPartyData returns a 100-byte party record. A boxed record is expanded by
// rebuilding its party-only fields from the encrypted data. The receiver is
// not mutated.
func (p *Pokemon) ToPartyData() ([]byte, error) {
	if len(p.RawData) == 100 {
		return append([]byte(nil), p.RawData...), nil
	}
	if len(p.RawData) != 80 {
		return nil, fmt.Errorf("gen 3 record is truncated")
	}
	out := make([]byte, 100)
	copy(out, p.RawData)
	if !p.ChecksumValid {
		return nil, fmt.Errorf("cannot expand a boxed Pokémon with an invalid checksum")
	}
	stats, err := p.partyStats()
	if err != nil {
		return nil, err
	}
	out[0x50] = 0 // status
	out[0x54] = p.Level
	out[0x55] = 0 // mail
	binary.LittleEndian.PutUint16(out[0x56:], stats[0])
	binary.LittleEndian.PutUint16(out[0x58:], stats[0])
	binary.LittleEndian.PutUint16(out[0x5A:], stats[1])
	binary.LittleEndian.PutUint16(out[0x5C:], stats[2])
	binary.LittleEndian.PutUint16(out[0x5E:], stats[3])
	binary.LittleEndian.PutUint16(out[0x60:], stats[4])
	binary.LittleEndian.PutUint16(out[0x62:], stats[5])
	return out, nil
}

// ToBoxData returns the 80-byte stored form. Party-only fields are dropped.
func (p *Pokemon) ToBoxData() ([]byte, error) {
	if len(p.RawData) == 80 {
		return append([]byte(nil), p.RawData...), nil
	}
	if len(p.RawData) != 100 {
		return nil, fmt.Errorf("gen 3 record is truncated")
	}
	return append([]byte(nil), p.RawData[:80]...), nil
}

// Encode builds a fresh 80- or 100-byte record from the struct's fields: it
// lays the four substructures out in the order implied by the personality
// value, writes the record checksum and encrypts the data block. A party
// record also carries the level and reconstructed stats.
func (p *Pokemon) Encode(size int) ([]byte, error) {
	if size != 80 && size != 100 {
		return nil, fmt.Errorf("gen 3 record must be 80 or 100 bytes, got %d", size)
	}
	raw := make([]byte, size)
	binary.LittleEndian.PutUint32(raw[0x00:], p.PersonalityValue)
	binary.LittleEndian.PutUint32(raw[0x04:], p.OTID)
	copy(raw[0x08:0x12], EncodeGen3String(p.Nickname, 10))
	raw[0x12] = p.Language
	raw[0x13] = p.Flags
	copy(raw[0x14:0x1B], EncodeGen3String(p.OTName, 7))
	raw[0x1B] = p.Markings

	key := p.PersonalityValue ^ p.OTID
	decrypted := make([]byte, 48)
	order := subStructOrder[p.PersonalityValue%24]
	b := &dataBlocks{decrypted: decrypted, key: key, order: [4]byte{order[0], order[1], order[2], order[3]}}
	for i, block := range order {
		blockData := decrypted[i*12 : i*12+12]
		switch block {
		case 'G':
			b.growth = blockData
		case 'A':
			b.attacks = blockData
		case 'E':
			b.evs = blockData
		case 'M':
			b.misc = blockData
		}
	}
	if b.growth == nil || b.attacks == nil || b.evs == nil || b.misc == nil {
		return nil, fmt.Errorf("gen 3 record has an invalid substructure order")
	}
	binary.LittleEndian.PutUint16(b.growth[0:2], p.Species)
	binary.LittleEndian.PutUint16(b.growth[2:4], p.HeldItem)
	binary.LittleEndian.PutUint32(b.growth[4:8], p.Experience)
	b.growth[8] = p.PPUps
	b.growth[9] = p.Friendship
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint16(b.attacks[i*2:], p.Moves[i])
		b.attacks[8+i] = p.PP[i]
	}
	copy(b.evs, p.EVs[:])
	b.misc[0] = p.Pokerus
	b.misc[1] = p.MetLocation
	origins := uint16(p.MetLevel) |
		uint16(p.OTGender&1)<<7 |
		uint16(p.OriginGame&0x0F)<<8 |
		uint16(p.BallID&0x0F)<<11
	binary.LittleEndian.PutUint16(b.misc[2:4], origins)
	ivWord := uint32(p.IVs[0]&0x1F) |
		uint32(p.IVs[1]&0x1F)<<5 |
		uint32(p.IVs[2]&0x1F)<<10 |
		uint32(p.IVs[3]&0x1F)<<15 |
		uint32(p.IVs[4]&0x1F)<<20 |
		uint32(p.IVs[5]&0x1F)<<25
	if p.IsEgg {
		ivWord |= 1 << 30
	}
	if p.AbilitySlot&1 == 1 {
		ivWord |= 1 << 31
	}
	binary.LittleEndian.PutUint32(b.misc[4:8], ivWord)

	p.RawData = raw
	p.writeBlocks(b)
	p.DataChecksum = p.calculateChecksum(decrypted)
	p.ChecksumValid = true

	if p.National != 0 && int(p.National) < len(pokemon.Growth3) {
		p.Level = pokemon.LevelFromExp(pokemon.Growth3[p.National], p.Experience)
	}
	if size == 100 {
		stats, err := p.partyStats()
		if err != nil {
			return nil, err
		}
		raw[0x50] = 0
		raw[0x54] = p.Level
		raw[0x55] = 0
		binary.LittleEndian.PutUint16(raw[0x56:], stats[0])
		binary.LittleEndian.PutUint16(raw[0x58:], stats[0])
		binary.LittleEndian.PutUint16(raw[0x5A:], stats[1])
		binary.LittleEndian.PutUint16(raw[0x5C:], stats[2])
		binary.LittleEndian.PutUint16(raw[0x5E:], stats[3])
		binary.LittleEndian.PutUint16(raw[0x60:], stats[4])
		binary.LittleEndian.PutUint16(raw[0x62:], stats[5])
	}
	return append([]byte(nil), raw...), nil
}

// partyStats returns HP, Atk, Def, Spe, SpA, SpD using Gen 3's stat formula.
func (p *Pokemon) partyStats() ([6]uint16, error) {
	if p.National == 0 || int(p.National) >= len(pokemon.BaseStats3) {
		return [6]uint16{}, fmt.Errorf("unknown species %d", p.Species)
	}
	base := pokemon.BaseStats3[p.National]
	level := int(p.Level)
	if level < 1 {
		level = 1
	}
	iv := p.IVs
	ev := p.EVs
	nature := int(p.Nature)
	up, down := nature/5, nature%5

	calc := func(base, iv, ev int, hp bool) int {
		value := ((2*base + iv + (ev / 4)) * level) / 100
		if hp {
			return value + level + 10
		}
		return value + 5
	}
	stats := [6]int{
		calc(base.HP, int(iv[0]), int(ev[0]), true),
		calc(base.Atk, int(iv[1]), int(ev[1]), false),
		calc(base.Def, int(iv[2]), int(ev[2]), false),
		calc(base.Spe, int(iv[3]), int(ev[3]), false),
		calc(base.SpA, int(iv[4]), int(ev[4]), false),
		calc(base.SpD, int(iv[5]), int(ev[5]), false),
	}
	if up != down {
		// The nature index is 0=Atk, 1=Def, 2=Spe, 3=SpA, 4=SpD, while the
		// stats array also holds HP at index 0, so shift by one.
		stats[up+1] = stats[up+1] * 110 / 100
		stats[down+1] = stats[down+1] * 90 / 100
	}
	var out [6]uint16
	for i, v := range stats {
		out[i] = uint16(v)
	}
	return out, nil
}

func GetParty(slot *SaveSlot) ([]*Pokemon, error) {
	if slot == nil {
		return nil, fmt.Errorf("active save slot is missing")
	}
	sec1, ok := slot.Sections[1]
	if !ok || len(sec1) < 0x38 {
		return nil, fmt.Errorf("section 1 missing")
	}

	teamSize := int(sec1[0x34])
	if teamSize > 6 {
		return nil, fmt.Errorf("invalid team size: %d", teamSize)
	}

	party := make([]*Pokemon, 0, teamSize)
	for i := 0; i < teamSize; i++ {
		offset := 0x38 + i*100
		if offset+100 > len(sec1) {
			return nil, fmt.Errorf("unexpected end of section 1 data")
		}
		poke, err := ParsePokemon(sec1[offset : offset+100])
		if err != nil {
			return nil, fmt.Errorf("party slot %d: %w", i+1, err)
		}
		if poke.Species == 0 {
			return nil, fmt.Errorf("party slot %d has species zero", i+1)
		}
		if !poke.ChecksumValid {
			return nil, fmt.Errorf("party slot %d has an invalid checksum", i+1)
		}
		party = append(party, poke)
	}
	return party, nil
}

func GetPCBoxes(slot *SaveSlot) ([]*Pokemon, error) {
	store := &slotStorage{save: &SaveFile{ActiveSlot: slot}}
	return store.boxPokemon()
}

// Implement core.Pokemon interface.

func (p *Pokemon) GetSpecies() uint16     { return p.Species }
func (p *Pokemon) GetNickname() string    { return p.Nickname }
func (p *Pokemon) GetLevel() byte         { return p.Level }
func (p *Pokemon) GetMoves() [4]uint16    { return p.Moves }
func (p *Pokemon) GetHeldItem() uint16    { return p.HeldItem }
func (p *Pokemon) GetTrainerName() string { return p.OTName }
func (p *Pokemon) GetTrainerID() uint32   { return p.OTID }
func (p *Pokemon) GetGeneration() int     { return 3 }
func (p *Pokemon) ToBytes() []byte        { return append([]byte(nil), p.RawData...) }

func (p *Pokemon) GetCaptureBall() (byte, bool) {
	return p.BallID, p.BallKnown
}
func (p *Pokemon) IsChecksumValid() bool { return p.ChecksumValid }

// TradeBlockReason rejects records that cannot safely participate in v1.
func (p *Pokemon) TradeBlockReason() string {
	if p.Species == 0 {
		return "the record has no species"
	}
	if p.National == 0 {
		return "the record's species has no National Dex mapping"
	}
	if p.Flags&0x04 != 0 {
		return "eggs cannot be traded"
	}
	if !p.ChecksumValid {
		return "the record checksum is invalid"
	}
	return ""
}

// NationalID returns the National Dex number, or 0 when the internal species
// ID has no National Dex mapping.
func (p *Pokemon) NationalID() uint16 { return p.National }

// DecodeGen3String converts a Gen 3 name field to text.
func DecodeGen3String(data []byte) string { return decodeGen3String(data) }

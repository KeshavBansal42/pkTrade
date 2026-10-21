package modern

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"pktrade/internal/core"
	"pktrade/internal/pokemon"
)

// Kind identifies the entity crypto used by a generation.
type Kind int

const (
	Kind45 Kind = iota // Gen 4/5: 32-byte blocks, checksum-seeded first 128 bytes
	Kind67             // Gen 6/7: 56-byte blocks
	Kind8              // Gen 8: 80-byte blocks
)

// Format describes the on-disk entity layout for a generation.
type Format struct {
	Generation int
	Kind       Kind
	StoredSize int
	PartySize  int
	BlockSize  int

	Species int
	Item    int
	ID32    int
	Exp     int
	PID     int
	Nature  int    // -1 means derived from PID
	EV      [6]int // HP, Atk, Def, Spe, SpA, SpD
	IV      int

	Nickname    int
	NicknameLen int
	OT          int
	OTLen       int

	// BallOffset is the byte that holds the capture ball of the record.
	// It is only meaningful while the record checksum is valid.
	BallOffset int

	Level int // -1 when the stored record has no level
	HP    int
	MaxHP int
	Atk   int
	Def   int
	Spe   int
	SpA   int
	SpD   int
	Tail  int // start of party-only bytes
}

var (
	Format4 = Format{
		Generation: 4, Kind: Kind45, StoredSize: 136, PartySize: 236, BlockSize: 32,
		Species: 0x08, Item: 0x0A, ID32: 0x0C, Exp: 0x10, PID: 0x00, Nature: -1,
		EV: [6]int{0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D}, IV: 0x38,
		Nickname: 0x48, NicknameLen: 22, OT: 0x68, OTLen: 16,
		BallOffset: 0x83,
		Level:      0x8C, HP: 0x8E, MaxHP: 0x90, Atk: 0x92, Def: 0x94, Spe: 0x96, SpA: 0x98, SpD: 0x9A,
		Tail: 136,
	}
	// Format4HGSS is the HeartGold/SoulSilver record layout. It is the same
	// as the Sinnoh layout except that a few original-trainer fields sit at
	// different offsets.
	Format4HGSS = Format{
		Generation: 4, Kind: Kind45, StoredSize: 136, PartySize: 236, BlockSize: 32,
		Species: 0x08, Item: 0x0A, ID32: 0x0C, Exp: 0x10, PID: 0x00, Nature: -1,
		EV: [6]int{0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D}, IV: 0x38,
		Nickname: 0x48, NicknameLen: 22, OT: 0x68, OTLen: 16,
		BallOffset: 0x86,
		Level:      0x8C, HP: 0x8E, MaxHP: 0x90, Atk: 0x92, Def: 0x94, Spe: 0x96, SpA: 0x98, SpD: 0x9A,
		Tail: 136,
	}
	Format5 = Format{
		Generation: 5, Kind: Kind45, StoredSize: 136, PartySize: 220, BlockSize: 32,
		Species: 0x08, Item: 0x0A, ID32: 0x0C, Exp: 0x10, PID: 0x00, Nature: 0x41,
		EV: [6]int{0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D}, IV: 0x38,
		Nickname: 0x48, NicknameLen: 22, OT: 0x68, OTLen: 16,
		BallOffset: 0x83,
		Level:      0x8C, HP: 0x8E, MaxHP: 0x90, Atk: 0x92, Def: 0x94, Spe: 0x96, SpA: 0x98, SpD: 0x9A,
		Tail: 136,
	}
	Format6 = Format{
		Generation: 6, Kind: Kind67, StoredSize: 0xE8, PartySize: 0x104, BlockSize: 56,
		Species: 0x08, Item: 0x0A, ID32: 0x0C, Exp: 0x10, PID: 0x18, Nature: 0x1C,
		EV: [6]int{0x1E, 0x1F, 0x20, 0x21, 0x22, 0x23}, IV: 0x74,
		Nickname: 0x40, NicknameLen: 26, OT: 0xB0, OTLen: 26,
		BallOffset: 0xDC,
		Level:      0xEC, HP: 0xF0, MaxHP: 0xF2, Atk: 0xF4, Def: 0xF6, Spe: 0xF8, SpA: 0xFA, SpD: 0xFC,
		Tail: 0xE8,
	}
	Format7 = Format{
		Generation: 7, Kind: Kind67, StoredSize: 0xE8, PartySize: 0x104, BlockSize: 56,
		Species: 0x08, Item: 0x0A, ID32: 0x0C, Exp: 0x10, PID: 0x18, Nature: 0x1C,
		EV: [6]int{0x1E, 0x1F, 0x20, 0x21, 0x22, 0x23}, IV: 0x74,
		Nickname: 0x40, NicknameLen: 26, OT: 0xB0, OTLen: 26,
		BallOffset: 0xDC,
		Level:      0xEC, HP: 0xF0, MaxHP: 0xF2, Atk: 0xF4, Def: 0xF6, Spe: 0xF8, SpA: 0xFA, SpD: 0xFC,
		Tail: 0xE8,
	}
	Format8 = Format{
		Generation: 8, Kind: Kind8, StoredSize: 0x148, PartySize: 0x158, BlockSize: 80,
		Species: 0x08, Item: 0x0A, ID32: 0x0C, Exp: 0x10, PID: 0x1C, Nature: 0x20,
		EV: [6]int{0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B}, IV: 0x8C,
		Nickname: 0x58, NicknameLen: 26, OT: 0xF8, OTLen: 26,
		BallOffset: 0x124,
		Level:      0x148, HP: 0x14A, MaxHP: 0x14C, Atk: 0x14E, Def: 0x150, Spe: 0x152, SpA: 0x154, SpD: 0x156,
		Tail: 0x148,
	}
)

func (f Format) withGeneration(gen int) Format {
	switch gen {
	case 4:
		return Format4
	case 5:
		return Format5
	case 6:
		return Format6
	case 7:
		return Format7
	case 8:
		return Format8
	}
	return f
}

// FormatFor returns the entity layout for a generation.
func FormatFor(gen int) Format { return Format{}.withGeneration(gen) }

// Pokemon is a decrypted Gen 4-8 Pokémon record.
type Pokemon struct {
	Format Format
	Data   []byte // decrypted record, stored size or party size
	Stored bool

	SpeciesID   uint16
	HeldItem    uint16
	OTID        uint32
	Exp         uint32
	PID         uint32
	NatureVal   byte
	LevelVal    byte
	NicknameStr string
	OTNameStr   string
	IsEgg       bool

	BoxNum  int
	BoxSlot int
}

// Parse decrypts and parses a stored or party-sized entity.
func Parse(format Format, raw []byte) (*Pokemon, error) {
	if len(raw) != format.StoredSize && len(raw) != format.PartySize {
		return nil, fmt.Errorf("Gen %d record must be %d or %d bytes, got %d", format.Generation, format.StoredSize, format.PartySize, len(raw))
	}
	empty := true
	for _, b := range raw {
		if b != 0 {
			empty = false
			break
		}
	}
	if empty {
		// The games leave unused box slots as zeroes instead of encrypting
		// them, so an all-zero slot is an empty slot and not a broken record.
		return nil, fmt.Errorf("empty Gen %d slot", format.Generation)
	}
	// A slot that was used and then emptied keeps the remains of the record
	// it held, but the game clears the personality value and the checksum.
	// Such a slot is empty as well.
	if isBlankHeader(format, raw) {
		return nil, fmt.Errorf("empty Gen %d slot", format.Generation)
	}
	data := append([]byte(nil), raw...)
	if err := crypt(format, data, false); err != nil {
		return nil, err
	}
	want := checksum16(data[8:format.StoredSize])
	if got := binary.LittleEndian.Uint16(data[6:8]); got != want {
		return nil, fmt.Errorf("Gen %d record checksum mismatch: got %04X, want %04X", format.Generation, got, want)
	}
	p := &Pokemon{Format: format, Data: data, Stored: len(raw) == format.StoredSize}
	p.refresh()
	if p.SpeciesID == 0 {
		return nil, fmt.Errorf("empty Gen %d slot", format.Generation)
	}
	return p, nil
}

func (p *Pokemon) refresh() {
	d := p.Data
	f := p.Format
	p.SpeciesID = binary.LittleEndian.Uint16(d[f.Species:])
	p.HeldItem = binary.LittleEndian.Uint16(d[f.Item:])
	p.OTID = binary.LittleEndian.Uint32(d[f.ID32:])
	p.Exp = binary.LittleEndian.Uint32(d[f.Exp:])
	p.PID = binary.LittleEndian.Uint32(d[f.PID:])
	if f.Nature >= 0 {
		p.NatureVal = d[f.Nature]
	} else {
		p.NatureVal = byte(p.PID % 25)
	}
	iv := binary.LittleEndian.Uint32(d[f.IV:])
	p.IsEgg = (iv>>30)&1 == 1
	if len(d) >= f.Level && f.Level >= 0 {
		p.LevelVal = d[f.Level]
	}
	if p.LevelVal < 1 || p.LevelVal > 100 {
		p.LevelVal = pokemon.LevelFromExp(p.growth(), p.Exp)
	}
	p.NicknameStr = decodeString(f.Generation, d[f.Nickname:f.Nickname+f.NicknameLen])
	p.OTNameStr = decodeString(f.Generation, d[f.OT:f.OT+f.OTLen])
}

func (p *Pokemon) growth() byte {
	n := p.NationalID()
	switch p.Format.Generation {
	case 4:
		if int(n) < len(pokemon.Growth4) {
			return pokemon.Growth4[n]
		}
	case 5:
		if int(n) < len(pokemon.Growth5) {
			return pokemon.Growth5[n]
		}
	case 6:
		if int(n) < len(pokemon.Growth6) {
			return pokemon.Growth6[n]
		}
	case 7:
		if int(n) < len(pokemon.Growth7) {
			return pokemon.Growth7[n]
		}
	case 8:
		if int(n) < len(pokemon.Growth8) {
			return pokemon.Growth8[n]
		}
	}
	return 3
}

func (p *Pokemon) NationalID() uint16 {
	if p.SpeciesID == 0 {
		return 0
	}
	limit := 0
	switch p.Format.Generation {
	case 4:
		limit = 493
	case 5:
		limit = 649
	case 6:
		limit = 721
	case 7:
		limit = 807
	case 8:
		limit = 898
	}
	if int(p.SpeciesID) > limit {
		return 0
	}
	return p.SpeciesID
}

func (p *Pokemon) GetSpecies() uint16     { return p.SpeciesID }
func (p *Pokemon) GetNickname() string    { return p.NicknameStr }
func (p *Pokemon) GetLevel() byte         { return p.LevelVal }
func (p *Pokemon) GetHeldItem() uint16    { return p.HeldItem }
func (p *Pokemon) GetTrainerName() string { return p.OTNameStr }
func (p *Pokemon) GetTrainerID() uint32   { return p.OTID }
func (p *Pokemon) GetGeneration() int     { return p.Format.Generation }

// isBlankHeader reports whether a record carries the empty-slot marker: a
// cleared personality value and checksum with the leftover body still in place.
func isBlankHeader(format Format, raw []byte) bool {
	if format.PID+4 > len(raw) {
		return false
	}
	for _, b := range raw[format.PID : format.PID+4] {
		if b != 0 {
			return false
		}
	}
	return raw[6] == 0 && raw[7] == 0
}

// GetCaptureBall returns the ball the Pokémon was caught in. The byte is only
// trustworthy while the record checksum is valid, so an unreadable record
// reports no ball at all.
func (p *Pokemon) GetCaptureBall() (byte, bool) {
	if p.Format.BallOffset <= 0 || p.Format.BallOffset >= len(p.Data) {
		return 0, false
	}
	if !p.IsChecksumValid() {
		return 0, false
	}
	ball := p.Data[p.Format.BallOffset]
	if ball == 0 {
		return 0, false
	}
	return ball, true
}
func (p *Pokemon) ToBytes() []byte { return append([]byte(nil), p.Data...) }
func (p *Pokemon) IsChecksumValid() bool {
	return checksum16(p.Data[8:p.Format.StoredSize]) == binary.LittleEndian.Uint16(p.Data[6:8])
}
func (p *Pokemon) GetMoves() [4]uint16 {
	// Move offsets differ between generations; these are the common low offsets.
	off := 0x28
	if p.Format.Generation >= 6 {
		off = 0x5A
	}
	if p.Format.Generation >= 8 {
		off = 0x72
	}
	out := [4]uint16{}
	for i := 0; i < 4; i++ {
		out[i] = binary.LittleEndian.Uint16(p.Data[off+i*2:])
	}
	return out
}

func (p *Pokemon) TradeBlockReason() string {
	if p.SpeciesID == 0 {
		return "the record has no species"
	}
	if p.IsEgg {
		return "eggs cannot be traded"
	}
	if p.NationalID() == 0 {
		return "the record's species has no National Dex mapping for this generation"
	}
	if p.LevelVal == 0 || p.LevelVal > 100 {
		return "the record's level is invalid"
	}
	if !p.IsChecksumValid() {
		return "the record checksum is invalid"
	}
	return ""
}

// Clone returns an independent copy.
func (p *Pokemon) Clone() *Pokemon {
	out := *p
	out.Data = append([]byte(nil), p.Data...)
	return &out
}

// SetSpecies changes the species and recomputes party stats.
func (p *Pokemon) SetSpecies(national uint16) error {
	if national == 0 || int(national) > 898 {
		return fmt.Errorf("invalid National Dex species %d", national)
	}
	p.SpeciesID = national
	binary.LittleEndian.PutUint16(p.Data[p.Format.Species:], national)
	p.refresh()
	return p.RecomputeStats()
}

// SetItem changes the held item.
func (p *Pokemon) SetItem(item uint16) {
	p.HeldItem = item
	binary.LittleEndian.PutUint16(p.Data[p.Format.Item:], item)
}

// SetDefaultNicknameIfNeeded renames a Pokémon whose nickname is its old
// species name.
func (p *Pokemon) SetDefaultNicknameIfNeeded(oldNational, newNational uint16) {
	if p.NicknameStr == pokemon.Name(oldNational) {
		p.NicknameStr = pokemon.Name(newNational)
		copy(p.Data[p.Format.Nickname:p.Format.Nickname+p.Format.NicknameLen], encodeString(p.Format.Generation, p.NicknameStr, p.Format.NicknameLen))
	}
}

func (p *Pokemon) stats() ([6]uint16, error) {
	n := p.NationalID()
	if n == 0 || int(n) > 898 {
		return [6]uint16{}, fmt.Errorf("no base stats for species %d", p.SpeciesID)
	}
	var base pokemon.BaseStats
	switch p.Format.Generation {
	case 4:
		base = pokemon.BaseStats4[n]
	case 5:
		base = pokemon.BaseStats5[n]
	case 6:
		base = pokemon.BaseStats6[n]
	case 7:
		base = pokemon.BaseStats7[n]
	case 8:
		base = pokemon.BaseStats8[n]
	}
	level := int(p.LevelVal)
	if level < 1 {
		level = 1
	}
	if level > 100 {
		level = 100
	}
	iv := binary.LittleEndian.Uint32(p.Data[p.Format.IV:])
	ivs := [6]int{
		int(iv & 0x1F), int((iv >> 5) & 0x1F), int((iv >> 10) & 0x1F),
		int((iv >> 15) & 0x1F), int((iv >> 20) & 0x1F), int((iv >> 25) & 0x1F),
	}
	evs := [6]int{}
	for i := range evs {
		evs[i] = int(p.Data[p.Format.EV[i]])
	}
	calc := func(b, iv, ev int, hp bool) int {
		v := (2*b + iv + ev/4) * level / 100
		if hp {
			return v + level + 10
		}
		return v + 5
	}
	raw := [6]uint16{
		uint16(calc(base.HP, ivs[0], evs[0], true)),
		uint16(calc(base.Atk, ivs[1], evs[1], false)),
		uint16(calc(base.Def, ivs[2], evs[2], false)),
		uint16(calc(base.Spe, ivs[3], evs[3], false)),
		uint16(calc(base.SpA, ivs[4], evs[4], false)),
		uint16(calc(base.SpD, ivs[5], evs[5], false)),
	}
	// Natures were introduced in Gen 3; Gen 8 uses the same multipliers.
	nat := int(p.NatureVal) % 25
	up, down := nat/5, nat%5
	if up != down {
		for i := 1; i < 6; i++ {
			if i-1 == up {
				raw[i] = uint16(float64(raw[i]) * 1.1)
			} else if i-1 == down {
				raw[i] = uint16(float64(raw[i]) * 0.9)
			}
		}
	}
	return raw, nil
}

// RecomputeStats rebuilds party-only stats when the record has a party tail.
func (p *Pokemon) RecomputeStats() error {
	f := p.Format
	if len(p.Data) < f.PartySize {
		return nil
	}
	stats, err := p.stats()
	if err != nil {
		return err
	}
	oldMax := int(binary.LittleEndian.Uint16(p.Data[f.MaxHP:]))
	oldHP := int(binary.LittleEndian.Uint16(p.Data[f.HP:]))
	p.Data[f.Level] = p.LevelVal
	put := func(off int, v uint16) { binary.LittleEndian.PutUint16(p.Data[off:], v) }
	put(f.MaxHP, stats[0])
	put(f.Atk, stats[1])
	put(f.Def, stats[2])
	put(f.Spe, stats[3])
	put(f.SpA, stats[4])
	put(f.SpD, stats[5])
	hp := int(stats[0])
	if oldMax > 0 && oldHP > 0 {
		hp = oldHP * int(stats[0]) / oldMax
	}
	if hp > int(stats[0]) {
		hp = int(stats[0])
	}
	put(f.HP, uint16(hp))
	return nil
}

// Encode returns an encrypted record for the requested size (stored or party).
func (p *Pokemon) Encode(size int) ([]byte, error) {
	f := p.Format
	if size != f.StoredSize && size != f.PartySize {
		return nil, fmt.Errorf("invalid Gen %d target size %d", f.Generation, size)
	}
	if len(p.Data) == size {
		out := append([]byte(nil), p.Data...)
		binary.LittleEndian.PutUint16(out[6:], checksum16(out[8:f.StoredSize]))
		if err := crypt(f, out, true); err != nil {
			return nil, err
		}
		return out, nil
	}
	if len(p.Data) == f.StoredSize && size == f.PartySize {
		out := make([]byte, f.PartySize)
		copy(out, p.Data)
		tmp := &Pokemon{Format: f, Data: out, Stored: false}
		tmp.refresh()
		if err := tmp.RecomputeStats(); err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint16(out[6:], checksum16(out[8:f.StoredSize]))
		if err := crypt(f, out, true); err != nil {
			return nil, err
		}
		return out, nil
	}
	if len(p.Data) == f.PartySize && size == f.StoredSize {
		out := append([]byte(nil), p.Data[:f.StoredSize]...)
		binary.LittleEndian.PutUint16(out[6:], checksum16(out[8:f.StoredSize]))
		if err := crypt(f, out, true); err != nil {
			return nil, err
		}
		return out, nil
	}
	return nil, fmt.Errorf("cannot convert Gen %d record from %d to %d bytes", f.Generation, len(p.Data), size)
}

// EncryptRecord recomputes the checksum and encrypts a decrypted record.
func EncryptRecord(f Format, data []byte) []byte {
	out := append([]byte(nil), data...)
	binary.LittleEndian.PutUint16(out[6:], checksum16(out[8:f.StoredSize]))
	_ = crypt(f, out, true)
	return out
}

// checksum16 is the sum of little-endian 16-bit words.
func checksum16(data []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(data); i += 2 {
		sum += uint32(binary.LittleEndian.Uint16(data[i:]))
	}
	return uint16(sum)
}

func crypt(f Format, data []byte, encrypt bool) error {
	if len(data) < f.StoredSize {
		return fmt.Errorf("record too short")
	}
	pv := binary.LittleEndian.Uint32(data[0:4])
	chk := binary.LittleEndian.Uint16(data[6:8])
	perm := blockPermutation(int((pv>>13)&31), encrypt)
	if encrypt {
		shuffle(data, f.BlockSize, perm)
	}
	// Gen 4 and Gen 5 seed the stored part with the record checksum, while
	// Gen 6 onwards use the personality value.
	seed := pv
	if f.Kind == Kind45 {
		seed = uint32(chk)
	}
	cryptArray(data, 8, f.StoredSize-8, seed)
	if len(data) > f.StoredSize {
		cryptArray(data, f.StoredSize, len(data)-f.StoredSize, pv)
	}
	if !encrypt {
		shuffle(data, f.BlockSize, perm)
	}
	return nil
}

func cryptArray(data []byte, offset, length int, seed uint32) {
	for i := offset; i+1 < offset+length; i += 2 {
		seed = 0x41C64E6D*seed + 0x6073
		binary.LittleEndian.PutUint16(data[i:], binary.LittleEndian.Uint16(data[i:])^uint16(seed>>16))
	}
}

// blockPermutation returns the four block indexes for a shuffle value. The
// games use the direct table when reading and the inverted table when
// writing, so the same bytes survive a write/read cycle.
func blockPermutation(sv int, encrypt bool) []byte {
	if !encrypt {
		return blockPosition[sv*4 : sv*4+4]
	}
	index := int(blockPositionInvert[sv])
	return blockPosition[index*4 : index*4+4]
}

func shuffle(data []byte, blockSize int, order []byte) {
	if len(order) == 0 {
		return
	}
	start := 8
	blocks := make([][]byte, len(order))
	for i := range blocks {
		blocks[i] = append([]byte(nil), data[start+i*blockSize:start+(i+1)*blockSize]...)
	}
	for i, src := range order {
		copy(data[start+i*blockSize:], blocks[src])
	}
}

var blockPosition = [...]byte{
	0, 1, 2, 3, 0, 1, 3, 2, 0, 2, 1, 3, 0, 3, 1, 2,
	0, 2, 3, 1, 0, 3, 2, 1, 1, 0, 2, 3, 1, 0, 3, 2,
	2, 0, 1, 3, 3, 0, 1, 2, 2, 0, 3, 1, 3, 0, 2, 1,
	1, 2, 0, 3, 1, 3, 0, 2, 2, 1, 0, 3, 3, 1, 0, 2,
	2, 3, 0, 1, 3, 2, 0, 1, 1, 2, 3, 0, 1, 3, 2, 0,
	2, 1, 3, 0, 3, 1, 2, 0, 2, 3, 1, 0, 3, 2, 1, 0,
	0, 1, 2, 3, 0, 1, 3, 2, 0, 2, 1, 3, 0, 3, 1, 2,
	0, 2, 3, 1, 0, 3, 2, 1, 1, 0, 2, 3, 1, 0, 3, 2,
}

var blockPositionInvert = [...]byte{
	0, 1, 2, 4, 3, 5, 6, 7, 12, 18, 13, 19, 8, 10, 14, 20,
	16, 22, 9, 11, 15, 21, 17, 23, 0, 1, 2, 4, 3, 5, 6, 7,
}

// DecodeString converts a Gen 4-8 name field to text.
func DecodeString(gen int, data []byte) string { return decodeString(gen, data) }

// EncodeString converts text to a Gen 4-8 name field.
func EncodeString(gen int, value string, length int) []byte { return encodeString(gen, value, length) }

func decodeString(gen int, data []byte) string {
	if gen >= 5 {
		var b strings.Builder
		for i := 0; i+1 < len(data); i += 2 {
			v := binary.LittleEndian.Uint16(data[i:])
			if v == 0 || v == 0xFFFF {
				break
			}
			b.WriteRune(rune(v))
		}
		return b.String()
	}
	var b strings.Builder
	for i := 0; i+1 < len(data); i += 2 {
		v := binary.LittleEndian.Uint16(data[i:])
		if v == 0 || v == 0xFFFF {
			break
		}
		if s, ok := gen4Charset[v]; ok {
			b.WriteString(s)
		}
	}
	return b.String()
}

func encodeString(gen int, value string, length int) []byte {
	out := make([]byte, length)
	if gen >= 5 {
		for i, r := range value {
			if i*2+1 >= length {
				break
			}
			binary.LittleEndian.PutUint16(out[i*2:], uint16(r))
		}
		return out
	}
	for i, r := range value {
		if i*2+1 >= length {
			break
		}
		binary.LittleEndian.PutUint16(out[i*2:], gen4Encode(r))
	}
	return out
}

var gen4Charset = func() map[uint16]string {
	m := map[uint16]string{
		0x120: "&", 0x1AC: "?", 0x1AD: "!", 0x1AE: "?", 0x1AF: ",", 0x1B0: ".",
		0x1B2: "/", 0x1B3: "'", 0x1B7: "(", 0x1B8: ")", 0x1BA: "+", 0x1BB: "-",
		0x1BC: "*", 0x1BD: "#", 0x1BE: "=", 0x1BF: "&", 0x1C0: "~", 0x1C1: ":",
		0x1C2: ";", 0x1D0: "@", 0x1D2: "%", 0x1DE: " ", 0x1B4: "’",
	}
	for i := 0; i < 10; i++ {
		m[uint16(0x121+i)] = string(rune('0' + i))
	}
	for i := 0; i < 26; i++ {
		m[uint16(0x12B+i)] = string(rune('A' + i))
		m[uint16(0x145+i)] = string(rune('a' + i))
	}
	return m
}()

func gen4Encode(r rune) uint16 {
	switch {
	case r == ' ':
		return 0x1DE
	case r >= '0' && r <= '9':
		return uint16(0x121 + (r - '0'))
	case r >= 'A' && r <= 'Z':
		return uint16(0x12B + (r - 'A'))
	case r >= 'a' && r <= 'z':
		return uint16(0x145 + (r - 'a'))
	}
	switch r {
	case '&':
		return 0x120
	case '?':
		return 0x1AC
	case '!':
		return 0x1AD
	case ',':
		return 0x1AF
	case '.':
		return 0x1B0
	case '/':
		return 0x1B2
	case '\'':
		return 0x1B3
	case '(':
		return 0x1B7
	case ')':
		return 0x1B8
	case '+':
		return 0x1BA
	case '-':
		return 0x1BB
	case '*':
		return 0x1BC
	case '#':
		return 0x1BD
	case '=':
		return 0x1BE
	case '~':
		return 0x1C0
	case ':':
		return 0x1C1
	case ';':
		return 0x1C2
	case '@':
		return 0x1D0
	case '%':
		return 0x1D2
	}
	return 0x1AC
}

var _ = core.Pokemon((*Pokemon)(nil))
var _ = math.Sqrt

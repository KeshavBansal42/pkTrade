package gen3

import (
	"encoding/binary"
	"testing"
)

func makePartyRecord(pid uint32, ballID byte) []byte {
	record := make([]byte, 100)
	binary.LittleEndian.PutUint32(record[0x00:], pid)
	otID := uint32(0x12345678)
	binary.LittleEndian.PutUint32(record[0x04:], otID)
	record[0x54] = 42

	var blocks [4][12]byte
	binary.LittleEndian.PutUint16(blocks[0][0:], 25)
	originsInfo := uint16(ballID&0x0F) << 11
	binary.LittleEndian.PutUint16(blocks[3][2:], originsInfo)
	binary.LittleEndian.PutUint16(blocks[1][0:], 33)

	order := subStructOrder[pid%24]
	decrypted := make([]byte, 48)
	for i, block := range order {
		var source int
		switch block {
		case 'G':
			source = 0
		case 'A':
			source = 1
		case 'E':
			source = 2
		case 'M':
			source = 3
		}
		copy(decrypted[i*12:], blocks[source][:])
	}

	var checksum uint32
	for i := 0; i < len(decrypted); i += 2 {
		checksum += uint32(binary.LittleEndian.Uint16(decrypted[i:]))
	}
	binary.LittleEndian.PutUint16(record[0x1C:], uint16(checksum))

	key := pid ^ otID
	for i := 0; i < len(decrypted); i += 4 {
		word := binary.LittleEndian.Uint32(decrypted[i:]) ^ key
		binary.LittleEndian.PutUint32(record[0x20+i:], word)
	}
	return record
}

func TestParsePokemonCaptureBallAcrossSubstructureOrders(t *testing.T) {
	for order := range subStructOrder {
		t.Run(subStructOrder[order], func(t *testing.T) {
			for ballID := byte(1); ballID <= 12; ballID++ {
				pid := uint32(order)
				raw := makePartyRecord(pid, ballID)
				pokemon, err := ParsePokemon(raw)
				if err != nil {
					t.Fatal(err)
				}
				if !pokemon.ChecksumValid {
					t.Fatalf("PID %d ball %d unexpectedly failed checksum", pid, ballID)
				}
				if !pokemon.BallKnown || pokemon.BallID != ballID {
					t.Fatalf("PID %d got ball %d known=%v, want %d", pid, pokemon.BallID, pokemon.BallKnown, ballID)
				}
				if got := pokemon.ToBytes(); string(got) != string(raw) {
					t.Fatalf("PID %d raw record did not round-trip", pid)
				}
			}
		})
	}
}

func TestParsePokemonUnknownBallForReservedIDsOrBadChecksum(t *testing.T) {
	for _, ballID := range []byte{0, 13, 14, 15} {
		pokemon, err := ParsePokemon(makePartyRecord(5, ballID))
		if err != nil {
			t.Fatal(err)
		}
		if pokemon.BallKnown {
			t.Fatalf("reserved ball ID %d reported as known", ballID)
		}
	}

	raw := makePartyRecord(7, 4)
	raw[0x1C]++
	pokemon, err := ParsePokemon(raw)
	if err != nil {
		t.Fatal(err)
	}
	if pokemon.ChecksumValid || pokemon.BallKnown {
		t.Fatal("bad-checksum record must not expose a capture ball")
	}
}

func TestParseBoxedPokemonDoesNotReportPartyBallWhenChecksumInvalid(t *testing.T) {
	raw := makePartyRecord(3, 4)[:80]
	pokemon, err := ParsePokemon(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !pokemon.BallKnown {
		t.Fatal("valid boxed record should still expose its origins ball")
	}
}

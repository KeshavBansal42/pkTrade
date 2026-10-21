package modern

import (
	"encoding/binary"
	"testing"
)

// TestCaptureBallReadsTheRightByte checks that every generation reads the
// capture ball from the byte the games use, and only when the record checksum
// is valid.
func TestCaptureBallReadsTheRightByte(t *testing.T) {
	cases := []struct {
		name   string
		format Format
		offset int
	}{
		{"gen4", Format4, 0x83},
		{"gen4 hgss", Format4HGSS, 0x86},
		{"gen5", Format5, 0x83},
		{"gen6", Format6, 0xDC},
		{"gen7", Format7, 0xDC},
		{"gen8", Format8, 0x124},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := make([]byte, tc.format.PartySize)
			binary.LittleEndian.PutUint32(raw[tc.format.PID:], 0x12345678)
			binary.LittleEndian.PutUint16(raw[tc.format.Species:], 25)
			raw[tc.format.BallOffset] = 4 // Poké Ball
			encrypted := EncryptRecord(tc.format, raw)

			p, err := Parse(tc.format, encrypted)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			ball, known := p.GetCaptureBall()
			if !known {
				t.Fatal("the capture ball was not reported")
			}
			if ball != 4 {
				t.Errorf("ball = %d, want 4", ball)
			}

			// A record with a broken checksum must not report a ball.
			broken := append([]byte(nil), encrypted...)
			broken[6] ^= 0xFF
			if _, err := Parse(tc.format, broken); err == nil {
				t.Fatal("expected a checksum error")
			}
		})
	}
}

func TestCaptureBallIsAbsentWhenUnset(t *testing.T) {
	raw := make([]byte, Format6.PartySize)
	binary.LittleEndian.PutUint32(raw[Format6.PID:], 0x12345678)
	binary.LittleEndian.PutUint16(raw[Format6.Species:], 25)
	encrypted := EncryptRecord(Format6, raw)
	p, err := Parse(Format6, encrypted)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ball, known := p.GetCaptureBall(); known {
		t.Errorf("a record without a ball reported %d", ball)
	}
}

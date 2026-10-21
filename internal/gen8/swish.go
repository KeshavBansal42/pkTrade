package gen8

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const (
	scObject = 4
	scArray  = 5
)

// Constants copied from the reference SwishCrypto implementation.
var introHash = []byte{
	0x9E, 0xC9, 0x9C, 0xD7, 0x0E, 0xD3, 0x3C, 0x44, 0xFB, 0x93, 0x03, 0xDC,
	0xEB, 0x39, 0xB4, 0x2A, 0x19, 0x47, 0xE9, 0x63, 0x4B, 0xA2, 0x33, 0x44,
	0x16, 0xBF, 0x82, 0xA2, 0xBA, 0x63, 0x55, 0xB6, 0x3D, 0x9D, 0xF2, 0x4B,
	0x5F, 0x7B, 0x6A, 0xB2, 0x62, 0x1D, 0xC2, 0x1B, 0x68, 0xE5, 0xC8, 0xB5,
	0x3A, 0x05, 0x90, 0x00, 0xE8, 0xA8, 0x10, 0x3D, 0xE2, 0xEC, 0xF0, 0x0C,
	0xB2, 0xED, 0x4F, 0x6D,
}

var outroHash = []byte{
	0xD6, 0xC0, 0x1C, 0x59, 0x8B, 0xC8, 0xB8, 0xCB, 0x46, 0xE1, 0x53, 0xFC,
	0x82, 0x8C, 0x75, 0x75, 0x13, 0xE0, 0x45, 0xDF, 0x32, 0x69, 0x3C, 0x75,
	0xF0, 0x59, 0xF8, 0xD9, 0xA2, 0x5F, 0xB2, 0x17, 0xE0, 0x80, 0x52, 0xDB,
	0xEA, 0x89, 0x73, 0x99, 0x75, 0x79, 0xAF, 0xCB, 0x2E, 0x80, 0x07, 0xE6,
	0xF1, 0x26, 0xE0, 0x03, 0x0A, 0xE6, 0x6F, 0xF6, 0x41, 0xBF, 0x7E, 0x59,
	0xC2, 0xAE, 0x55, 0xFD,
}

var xorPad = []byte{
	0xA0, 0x92, 0xD1, 0x06, 0x07, 0xDB, 0x32, 0xA1, 0xAE, 0x01, 0xF5, 0xC5,
	0x1E, 0x84, 0x4F, 0xE3, 0x53, 0xCA, 0x37, 0xF4, 0xA7, 0xB0, 0x4D, 0xA0,
	0x18, 0xB7, 0xC2, 0x97, 0xDA, 0x5F, 0x53, 0x2B, 0x75, 0xFA, 0x48, 0x16,
	0xF8, 0xD4, 0x8A, 0x6F, 0x61, 0x05, 0xF4, 0xE2, 0xFD, 0x04, 0xB5, 0xA3,
	0x0F, 0xFC, 0x44, 0x92, 0xCB, 0x32, 0xE6, 0x1B, 0xB9, 0xB1, 0x2E, 0x01,
	0xB0, 0x56, 0x53, 0x36, 0xD2, 0xD1, 0x50, 0x3D, 0xDE, 0x5B, 0x2E, 0x0E,
	0x52, 0xFD, 0xDF, 0x2F, 0x7B, 0xCA, 0x63, 0x50, 0xA4, 0x67, 0x5D, 0x23,
	0x17, 0xC0, 0x52, 0xE1, 0xA6, 0x30, 0x7C, 0x2B, 0xB6, 0x70, 0x36, 0x5B,
	0x2A, 0x27, 0x69, 0x33, 0xF5, 0x63, 0x7B, 0x36, 0x3F, 0x26, 0x9B, 0xA3,
	0xED, 0x7A, 0x53, 0x00, 0xA4, 0x48, 0xB3, 0x50, 0x9E, 0x14, 0xA0, 0x52,
	0xDE, 0x7E, 0x10, 0x2B, 0x1B, 0x77, 0x6E,
}

type scBlock struct {
	Key  uint32
	Type byte
	Sub  byte
	Data []byte
}

func (b *scBlock) size() int {
	n := 5 + len(b.Data)
	if b.Type == scObject {
		n += 4
	}
	if b.Type == scArray {
		n += 5
	}
	return n
}

type xorshift32 struct {
	state   uint32
	counter int
}

func newXorShift32(seed uint32) *xorshift32 {
	state := seed
	steps := popcount(state)
	for i := 0; i < steps; i++ {
		state = advance(state)
	}
	return &xorshift32{state: state}
}

func advance(state uint32) uint32 {
	state ^= state << 2
	state ^= state >> 15
	state ^= state << 13
	return state
}

func popcount(v uint32) int {
	n := 0
	for v != 0 {
		n += int(v & 1)
		v >>= 1
	}
	return n
}

func (x *xorshift32) next() byte {
	b := byte(x.state >> (x.counter * 8))
	if x.counter == 3 {
		x.state = advance(x.state)
		x.counter = 0
	} else {
		x.counter++
	}
	return b
}

func (x *xorshift32) next32() uint32 {
	return uint32(x.next()) | uint32(x.next())<<8 | uint32(x.next())<<16 | uint32(x.next())<<24
}

func (x *xorshift32) crypt(data []byte) {
	for i := range data {
		data[i] ^= x.next()
	}
}

func hashValid(data []byte) bool {
	if len(data) <= 32 {
		return false
	}
	h := sha256.New()
	h.Write(introHash)
	h.Write(data[:len(data)-32])
	h.Write(outroHash)
	sum := h.Sum(nil)
	return string(sum) == string(data[len(data)-32:])
}

func computeHash(payload []byte) []byte {
	h := sha256.New()
	h.Write(introHash)
	h.Write(payload)
	h.Write(outroHash)
	return h.Sum(nil)
}

func xorPayload(data []byte) []byte {
	out := append([]byte(nil), data...)
	for i := range out {
		out[i] ^= xorPad[i%len(xorPad)]
	}
	return out
}

func decryptSwitch(data []byte) ([]*scBlock, error) {
	if !hashValid(data) {
		return nil, fmt.Errorf("Gen 8 save hash mismatch")
	}
	payload := xorPayload(data[:len(data)-32])
	blocks := make([]*scBlock, 0, 256)
	for off := 0; off < len(payload); {
		if off+5 > len(payload) {
			return nil, fmt.Errorf("truncated Gen 8 block header at 0x%X", off)
		}
		key := binary.LittleEndian.Uint32(payload[off:])
		off += 4
		xk := newXorShift32(key)
		typ := payload[off] ^ xk.next()
		off++
		b := &scBlock{Key: key, Type: typ}
		switch typ {
		case 1, 2, 3:
			blocks = append(blocks, b)
			continue
		case scObject:
			if off+4 > len(payload) {
				return nil, fmt.Errorf("truncated object block")
			}
			count := int(binary.LittleEndian.Uint32(payload[off:]) ^ xk.next32())
			off += 4
			if count < 0 || off+count > len(payload) {
				return nil, fmt.Errorf("invalid object length %d", count)
			}
			b.Data = append([]byte(nil), payload[off:off+count]...)
			xk.crypt(b.Data)
			off += count
		case scArray:
			if off+5 > len(payload) {
				return nil, fmt.Errorf("truncated array block")
			}
			entries := int(binary.LittleEndian.Uint32(payload[off:]) ^ xk.next32())
			off += 4
			sub := payload[off] ^ xk.next()
			off++
			sz := typeSize(sub)
			if entries < 0 || sz <= 0 || off+entries*sz > len(payload) {
				return nil, fmt.Errorf("invalid array length %d", entries)
			}
			b.Sub = sub
			b.Data = append([]byte(nil), payload[off:off+entries*sz]...)
			xk.crypt(b.Data)
			off += entries * sz
		default:
			sz := typeSize(typ)
			if sz <= 0 || off+sz > len(payload) {
				return nil, fmt.Errorf("invalid scalar block type %d", typ)
			}
			b.Data = append([]byte(nil), payload[off:off+sz]...)
			xk.crypt(b.Data)
			off += sz
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

func typeSize(t byte) int {
	switch t {
	case 1, 2, 3, 8, 12:
		return 1
	case 9, 13:
		return 2
	case 10, 14, 16:
		return 4
	case 11, 15, 17:
		return 8
	}
	return 0
}

func encryptSwitch(blocks []*scBlock) []byte {
	out := make([]byte, 0, 1024*1024)
	for _, b := range blocks {
		xk := newXorShift32(b.Key)
		var hdr [5]byte
		binary.LittleEndian.PutUint32(hdr[0:], b.Key)
		hdr[4] = b.Type ^ xk.next()
		out = append(out, hdr[:]...)
		switch b.Type {
		case 1, 2, 3:
			continue
		case scObject:
			var n [4]byte
			binary.LittleEndian.PutUint32(n[:], uint32(len(b.Data))^xk.next32())
			out = append(out, n[:]...)
		case scArray:
			var n [4]byte
			binary.LittleEndian.PutUint32(n[:], uint32(len(b.Data)/typeSize(b.Sub))^xk.next32())
			out = append(out, n[:]...)
			out = append(out, b.Sub^xk.next())
		}
		payload := append([]byte(nil), b.Data...)
		xk.crypt(payload)
		out = append(out, payload...)
	}
	out = xorPayload(out)
	return append(out, computeHash(out)...)
}

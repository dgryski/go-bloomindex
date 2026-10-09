package bloomindex

import (
	"math/rand"
	"testing"
	"unsafe"
)

// rowsAt returns n bitrows backed by a []uint64 whose start address is
// offset bytes past a 16-byte boundary.  bitrow only requires 8-byte
// alignment, so queryCore must not assume the rows are 16-byte aligned.
func rowsAt(n int, offset uintptr) []bitrow {
	buf := make([]uint64, 8*n+2)
	i := 0
	for (uintptr(unsafe.Pointer(&buf[i])) % 16) != offset {
		i++
	}
	return unsafe.Slice((*bitrow)(unsafe.Pointer(&buf[i])), n)
}

func queryRef(bits []bitrow, hashes []uint32) bitrow {
	r := bitrow{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}
	for _, h := range hashes {
		var any uint64
		for j := range r {
			r[j] &= bits[h][j]
			any |= r[j]
		}
		if any == 0 {
			break
		}
	}
	return r
}

func TestQueryCoreAlignment(t *testing.T) {
	const nrows = 64

	tests := []struct {
		name   string
		offset uintptr
	}{
		{"aligned16", 0},
		{"aligned8", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rnd := rand.New(rand.NewSource(1))
			bits := rowsAt(nrows, tt.offset)
			for i := range bits {
				for j := range bits[i] {
					// mostly-set bits so queries don't terminate immediately
					bits[i][j] = rnd.Uint64() | rnd.Uint64() | rnd.Uint64()
				}
			}

			for iter := 0; iter < 100; iter++ {
				hashes := make([]uint32, 1+rnd.Intn(8))
				for i := range hashes {
					hashes[i] = uint32(rnd.Intn(nrows))
				}

				var got bitrow
				queryCore(&got, bits, hashes)
				if want := queryRef(bits, hashes); got != want {
					t.Fatalf("queryCore(%v) = %x, want %x", hashes, got, want)
				}
			}
		})
	}
}

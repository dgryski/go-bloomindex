package bloomindex

import (
	"math/rand"
	"strconv"
	"testing"
)

func BenchmarkQueryCore(b *testing.B) {
	for _, k := range []int{4, 16} {
		rnd := rand.New(rand.NewSource(1))
		bits := make([]bitrow, 4096)
		for i := range bits {
			for j := range bits[i] {
				// clear one bit per word so queries don't terminate early
				bits[i][j] = ^uint64(0) ^ (1 << uint(rnd.Intn(64)))
			}
		}
		hashes := make([]uint32, k)
		for i := range hashes {
			hashes[i] = uint32(rnd.Intn(len(bits)))
		}

		b.Run("hashes="+strconv.Itoa(k), func(b *testing.B) {
			var r bitrow
			for i := 0; i < b.N; i++ {
				queryCore(&r, bits, hashes)
			}
		})
	}
}

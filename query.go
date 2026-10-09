//go:build !amd64 || purego
// +build !amd64 purego

package bloomindex

//gc:nosplit
func queryCore(r *bitrow, bits []bitrow, hashes []uint32) {

	// Accumulate in locals rather than through r, and take a single pointer
	// to each row, so the compiler can keep everything in registers and
	// bounds-check bits[bit] once per hash instead of once per word.
	r0, r1, r2, r3 := ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)
	r4, r5, r6, r7 := ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)

	for _, bit := range hashes {
		row := &bits[bit]
		r0 &= row[0]
		r1 &= row[1]
		r2 &= row[2]
		r3 &= row[3]
		r4 &= row[4]
		r5 &= row[5]
		r6 &= row[6]
		r7 &= row[7]

		if (r0 | r1 | r2 | r3 | r4 | r5 | r6 | r7) == 0 {
			break
		}
	}

	*r = bitrow{r0, r1, r2, r3, r4, r5, r6, r7}
}

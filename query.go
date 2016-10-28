package bloomindex

//gc:nosplit
func queryCore(r *bitrow, bits []bitrow, hashes []uint16) {
	for _, bit := range hashes {
		r[0] &= bits[bit][0]
		r[1] &= bits[bit][1]
		r[2] &= bits[bit][2]
		r[3] &= bits[bit][3]
	}
}

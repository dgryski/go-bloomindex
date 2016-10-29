package bloomindex

//gc:nosplit
func queryCore(r *bitrow, bits []bitrow, hashes []uint16) {

	*r = bitrow{
		0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff,
		0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff,
	}

	for _, bit := range hashes {
		r[0] &= bits[bit][0]
		r[1] &= bits[bit][1]
		r[2] &= bits[bit][2]
		r[3] &= bits[bit][3]
		r[4] &= bits[bit][4]
		r[5] &= bits[bit][5]
		r[6] &= bits[bit][6]
		r[7] &= bits[bit][7]
	}
}

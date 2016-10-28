// Package bloomindex is a
package bloomindex

// http://bitfunnel.org/strangeloop/

import (
	"errors"
)

type DocID uint64

type Index struct {
	blocks []Block

	blockSize int
	hashes    uint16
	mask      uint16
}

func NewIndex(blockSize int, h int) *Index {
	return &Index{
		blockSize: blockSize,
		hashes:    uint16(h),
		mask:      uint16(blockSize) - 1,
	}
}

func (idx *Index) AddDocument() DocID {

	if len(idx.blocks) == 0 {
		idx.blocks = append(idx.blocks, newBlock(idx.blockSize))
	}

	blkid := len(idx.blocks) - 1
	if idx.blocks[blkid].numDocuments() == idsPerBlock {
		// full -- allocate a new one
		idx.blocks = append(idx.blocks, newBlock(idx.blockSize))
		blkid++
	}
	docid, _ := idx.blocks[blkid].addDocument()

	return DocID(uint64(blkid)*idsPerBlock + uint64(docid))
}

func (idx *Index) AddTerms(docid DocID, terms []uint32) {

	blkid := docid / idsPerBlock
	id := uint8(docid % idsPerBlock)

	for _, t := range terms {
		h := xorshift32(t)
		h1, h2 := uint16(h>>16), uint16(h)
		for i := uint16(0); i < idx.hashes; i++ {
			idx.blocks[blkid].setbit(id, (h1+i*h2)&idx.mask)
		}
	}
}

func (idx *Index) Query(terms []uint32) []DocID {

	var docs []DocID

	var bits []uint16

	for _, t := range terms {
		h := xorshift32(t)
		h1, h2 := uint16(h>>16), uint16(h)
		for i := uint16(0); i < idx.hashes; i++ {
			bits = append(bits, (h1+i*h2)&idx.mask)
		}
	}

	for i, blk := range idx.blocks {
		d := blk.query(bits)
		for _, dd := range d {
			docs = append(docs, DocID(uint64(i*idsPerBlock)+uint64(dd)))
		}
	}

	return docs
}

const idsPerBlock = 64

type Block struct {
	bits []uint64

	// valid is the number of valid documents in this block
	// TODO(dgryski): upgrade to mask at some point
	valid uint64
}

func newBlock(size int) Block {
	return Block{
		bits: make([]uint64, size),
	}
}

func (b *Block) numDocuments() uint64 {
	return b.valid
}

var errNoSpace = errors.New("block: no space")

func (b *Block) addDocument() (uint64, error) {
	if b.valid == 64 {
		return 0, errNoSpace
	}

	docid := b.valid
	b.valid++
	return docid, nil
}

func (b *Block) setbit(docid uint8, bit uint16) {
	b.bits[bit] |= 1 << docid
}

func (b *Block) getbit(docid uint8, bit uint16) uint64 {
	return b.bits[bit] & (1 << docid)
}

func (b *Block) get(bit uint16) uint64 {
	return b.bits[bit]
}

func (b *Block) query(bits []uint16) []uint8 {

	if len(bits) == 0 {
		return nil
	}

	r := b.bits[bits[0]]

	for _, bit := range bits[1:] {
		r &= b.bits[bit]
	}

	// mask off the invalid documents
	r &= (1 << b.valid) - 1

	// return the IDs of the remaining
	return popset(r)
}

// popset returns which bits are set in r
func popset(u uint64) []uint8 {
	var r []uint8

	// TODO(dgryski): optimization opportunity
	for i := uint8(0); i < 64; i++ {
		if u&(1<<i) != 0 {
			r = append(r, i)
		}
	}

	return r
}

// Xorshift32 is an xorshift RNG
func xorshift32(y uint32) uint32 {

	// http://www.jstatsoft.org/v08/i14/paper
	// Marasaglia's "favourite"

	y ^= (y << 13)
	y ^= (y >> 17)
	y ^= (y << 5)
	return y
}

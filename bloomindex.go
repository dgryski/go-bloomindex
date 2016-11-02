// Package bloomindex is a
package bloomindex

// http://bitfunnel.org/strangeloop/

import (
	"errors"
	"github.com/dgryski/go-bits"
)

type DocID uint64

type Index struct {
	blocks []Block

	meta []Block

	blockSize int
	metaSize  int
	hashes    uint32
	mask      uint32
	mmask     uint32
}

const metaScale = 64

func NewIndex(blockSize, metaSize int, hashes int) *Index {
	return &Index{
		blockSize: blockSize,
		metaSize:  metaSize,
		hashes:    uint32(hashes),
		mask:      uint32(blockSize) - 1,
		mmask:     uint32(metaSize) - 1,
	}
}

func (idx *Index) AddDocument(terms []uint32) DocID {

	if len(idx.blocks) == 0 {
		idx.blocks = append(idx.blocks, newBlock(idx.blockSize))
		idx.meta = append(idx.meta, newBlock(idx.metaSize))
	}

	blkid := len(idx.blocks) - 1
	if idx.blocks[blkid].numDocuments() == idsPerBlock {
		// full -- allocate a new one
		idx.blocks = append(idx.blocks, newBlock(idx.blockSize))
		blkid++

		if idx.meta[len(idx.meta)-1].numDocuments() == idsPerBlock {
			idx.meta = append(idx.meta, newBlock(idx.metaSize))
		}
	}
	docid, _ := idx.blocks[blkid].addDocument()

	idx.meta[blkid/idsPerBlock].addDocument()

	extdocid := DocID(uint64(blkid)*idsPerBlock + uint64(docid))

	idx.AddTerms(extdocid, terms)

	return extdocid
}

func (idx *Index) AddTerms(docid DocID, terms []uint32) {

	blkid := docid / idsPerBlock
	id := uint16(docid % idsPerBlock)

	mblkid := blkid / idsPerBlock
	mid := uint16(blkid % idsPerBlock)

	for _, t := range terms {
		h1, h2 := xorshift32(t), jenkins32(t)
		for i := uint32(0); i < idx.hashes; i++ {
			idx.blocks[blkid].setbit(id, (h1+i*h2)&idx.mask)
			idx.meta[mblkid].setbit(mid, (h1+i*h2)&idx.mmask)
		}
	}
}

func (idx *Index) Query(terms []uint32) []DocID {

	var docs []DocID

	var bits []uint32
	var mbits []uint32

	for _, t := range terms {
		h1, h2 := xorshift32(t), jenkins32(t)
		for i := uint32(0); i < idx.hashes; i++ {
			bits = append(bits, (h1+i*h2)&idx.mask)
			mbits = append(mbits, (h1+i*h2)&idx.mmask)
		}
	}

	for i, mblk := range idx.meta {

		blks := mblk.query(mbits)

		for _, blkid := range blks {
			b := (i*idsPerBlock + int(blkid))
			blk := idx.blocks[b]

			d := blk.query(bits)
			for _, dd := range d {
				docs = append(docs, DocID(uint64(b*idsPerBlock)+uint64(dd)))
			}

		}
	}

	return docs
}

const idsPerBlock = 512

type bitrow [8]uint64

type Block struct {
	bits []bitrow

	// valid is the number of valid documents in this block
	// TODO(dgryski): upgrade to mask at some point
	valid uint64
}

func newBlock(size int) Block {
	return Block{
		bits: make([]bitrow, size),
	}
}

func (b *Block) numDocuments() uint64 {
	return b.valid
}

var errNoSpace = errors.New("block: no space")

func (b *Block) addDocument() (uint64, error) {
	if b.valid == idsPerBlock {
		return 0, errNoSpace
	}

	docid := b.valid
	b.valid++
	return docid, nil
}

func (b *Block) setbit(docid uint16, bit uint32) {
	b.bits[bit][docid>>6] |= 1 << (docid & 0x3f)
}

func (b *Block) getbit(docid uint16, bit uint32) uint64 {
	return b.bits[bit][docid>>6] & (1 << (docid & 0x3f))
}

func (b *Block) get(bit uint32) bitrow {
	return b.bits[bit]
}

func (b *Block) query(bits []uint32) []uint16 {

	if len(bits) == 0 {
		return nil
	}

	var r bitrow

	queryCore(&r, b.bits, bits)

	// return the IDs of the remaining
	return popset(r)
}

// popset returns which bits are set in r
func popset(b bitrow) []uint16 {
	var r []uint16

	var docid uint64
	for i, u := range b {
		docid = uint64(i) * 64
		for u != 0 {
			tz := bits.Ctz(u)
			u >>= tz + 1
			docid += tz
			r = append(r, uint16(docid))
			docid++
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

// jenkins32 is Robert Jenkins' 32-bit integer hash function
func jenkins32(a uint32) uint32 {
	a = (a + 0x7ed55d16) + (a << 12)
	a = (a ^ 0xc761c23c) ^ (a >> 19)
	a = (a + 0x165667b1) + (a << 5)
	a = (a + 0xd3a2646c) ^ (a << 9)
	a = (a + 0xfd7046c5) + (a << 3)
	a = (a ^ 0xb55a4f09) ^ (a >> 16)
	return a
}

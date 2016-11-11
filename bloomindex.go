// Package bloomindex is a bloom-filter based search index
/*

References:

    "Reasoning about performance (in the context of search)"
    Dan Luu
    http://bitfunnel.org/strangeloop/

    "Bloofi: Multidimensional Bloom Filters"
    Adina Crainiceanu, Daniel Lemire
    https://arxiv.org/abs/1501.01941

*/
package bloomindex

import (
	"errors"
	"github.com/dgryski/go-bits"
	"math"
)

type DocID uint64

type Index struct {
	blocks []block

	meta []block

	blockSize int
	metaSize  int
	hashes    uint32
	mask      uint32
	mmask     uint32
}

func NewIndex(blockSize, metaSize int, hashes int) *Index {
	idx := &Index{
		blocks:    []block{newBlock(blockSize)},
		meta:      []block{newBlock(metaSize)},
		blockSize: blockSize,
		metaSize:  metaSize,
		hashes:    uint32(hashes),
		mask:      uint32(blockSize) - 1,
		mmask:     uint32(metaSize) - 1,
	}

	// we start out with a single block in our meta index
	idx.meta[0].addDocument()

	return idx
}

func (idx *Index) AddDocument(terms []uint32) DocID {

	blockid := len(idx.blocks) - 1
	if idx.blocks[blockid].numDocuments() == idsPerBlock {
		// full -- allocate a new one
		idx.blocks = append(idx.blocks, newBlock(idx.blockSize))
		blockid++

		mblockid := len(idx.meta) - 1
		if idx.meta[mblockid].numDocuments() == idsPerBlock {
			idx.meta = append(idx.meta, newBlock(idx.metaSize))
			mblockid++
		}
		idx.meta[mblockid].addDocument()

	}
	docid, _ := idx.blocks[blockid].addDocument()

	idx.addTerms(blockid, uint16(docid), terms)

	return DocID(uint64(blockid)*idsPerBlock + uint64(docid))
}

func (idx *Index) addTerms(blockid int, docid uint16, terms []uint32) {

	mblockid := blockid / idsPerBlock
	mdocid := uint16(blockid % idsPerBlock)

	for _, t := range terms {
		h1, h2 := xorshift32(t), jenkins32(t)
		for i := uint32(0); i < idx.hashes; i++ {
			idx.blocks[blockid].setbit(docid, (h1+i*h2)&idx.mask)
			idx.meta[mblockid].setbit(mdocid, (h1+i*h2)&idx.mmask)
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

	var mblocks []uint16
	var bdocs []uint16
	for i, mblock := range idx.meta {

		mblocks = mblock.query(mbits, mblocks[:0])

		for _, blockid := range mblocks {
			b := (i*idsPerBlock + int(blockid))
			block := idx.blocks[b]

			d := block.query(bits, bdocs[:0])
			for _, dd := range d {
				docs = append(docs, DocID(uint64(b*idsPerBlock)+uint64(dd)))
			}

		}
	}

	return docs
}

type ShardedIndex struct {

	// sharded index
	idxs []Index

	// mapping from internal shard document ID to external
	docs [][]DocID

	documents DocID

	hashes int
	fprate float64
}

func NewShardedIndex(fprate float64, hashes int) *ShardedIndex {
	return &ShardedIndex{
		idxs:   make([]Index, 32),
		docs:   make([][]DocID, 32),
		fprate: fprate,
		hashes: hashes,
	}
}

func (sh *ShardedIndex) AddDocument(terms []uint32) DocID {

	u32terms := nextPowerOfTwo(uint32(len(terms)))

	shard := ilog2(u32terms)

	if sh.idxs[shard].meta == nil {
		// doesn't exist yet

		size := filterBits(int(u32terms), sh.fprate)
		if size < 128 {
			size = 128
		}
		sh.idxs[shard] = *NewIndex(int(size), int(size*idsPerBlock), sh.hashes)
	}

	sh.idxs[shard].AddDocument(terms)
	extid := sh.documents

	sh.docs[shard] = append(sh.docs[shard], extid)

	sh.documents++

	return extid
}

func (sh *ShardedIndex) Query(terms []uint32) []DocID {

	var docs []DocID

	for i := range sh.idxs {
		d := sh.idxs[i].Query(terms)
		for _, dd := range d {
			docs = append(docs, sh.docs[i][dd])
		}
	}

	return docs
}

// return the integer >= i which is a power of two
func nextPowerOfTwo(i uint32) uint32 {
	n := i - 1
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// integer log base 2
func ilog2(v uint32) uint64 {
	var r uint64
	for ; v != 0; v >>= 1 {
		r++
	}
	return r
}

// FilterBits returns the number of bits required for the desired capacity and
// false positive rate.
func filterBits(capacity int, falsePositiveRate float64) uint32 {
	bits := float64(capacity) * -math.Log(falsePositiveRate) / (math.Log(2.0) * math.Log(2.0)) // in bits
	m := nextPowerOfTwo(uint32(bits))

	return m
}

const idsPerBlock = 512

type bitrow [8]uint64

type block struct {
	bits []bitrow

	// valid is the number of valid documents in this block
	// TODO(dgryski): upgrade to mask at some point
	valid uint16
}

func newBlock(size int) block {
	return block{
		bits: make([]bitrow, size),
	}
}

func (b *block) numDocuments() uint16 {
	return b.valid
}

var errNoSpace = errors.New("block: no space")

func (b *block) addDocument() (uint16, error) {
	if b.valid == idsPerBlock {
		return 0, errNoSpace
	}

	docid := b.valid
	b.valid++
	return docid, nil
}

func (b *block) setbit(docid uint16, bit uint32) {
	b.bits[bit][docid>>6] |= 1 << (docid & 0x3f)
}

func (b *block) getbit(docid uint16, bit uint32) uint64 {
	return b.bits[bit][docid>>6] & (1 << (docid & 0x3f))
}

func (b *block) get(bit uint32) bitrow {
	return b.bits[bit]
}

func (b *block) query(bits []uint32, docs []uint16) []uint16 {

	if len(bits) == 0 {
		return nil
	}

	var r bitrow

	queryCore(&r, b.bits, bits)

	// return the IDs of the remaining
	return popset(r, docs)
}

// popset returns which bits are set in b
func popset(b bitrow, r []uint16) []uint16 {

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

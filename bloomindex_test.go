package bloomindex

import (
	"hash/crc32"
	"math/rand"
	"strings"
	"testing"
)

func TestBlockGetSet(t *testing.T) {

	bl := newBlock(64)
	bl.valid = 64

	set := make(map[uint64]bool)

	for i := 0; i < 1000; i++ {
		doc := uint32(rand.Intn(64))
		bit := uint32(rand.Intn(64))

		set[uint64(doc)<<32+uint64(bit)] = true

		bl.setbit(uint16(doc), bit)
	}

	for doc := uint32(0); doc < 64; doc++ {
		for bit := uint32(0); bit < 64; bit++ {
			want := set[uint64(doc)<<32+uint64(bit)]

			got := bl.getbit(uint16(doc), bit) != 0

			if want != got {
				t.Errorf("bl.get(%d,%d)=%v, want %v", doc, bit, got, want)
			}
		}
	}
}

func TestBlockQuery(t *testing.T) {
	// from the paper

	var bits = []struct {
		docs []uint16
	}{
		0:  {[]uint16{'A'}},
		1:  {[]uint16{'F', 'I', 'J'}},
		2:  {[]uint16{'H'}},
		3:  {[]uint16{'G', 'J'}},
		4:  {[]uint16{'I'}},
		5:  {[]uint16{'I', 'J'}},
		6:  {[]uint16{'E', 'H'}},
		7:  {[]uint16{'F', 'I', 'J'}},
		8:  {nil},
		9:  {[]uint16{'C'}},
		10: {[]uint16{'J'}},
		11: {[]uint16{'B', 'D'}},
		12: {[]uint16{'D', 'I', 'J'}},
		13: {[]uint16{'B'}},
		14: {nil},
		15: {[]uint16{'G', 'H'}},
	}

	bl := newBlock(16)
	bl.valid = 10

	for i, b := range bits {
		for _, d := range b.docs {
			bl.setbit(d-'A', uint32(i))
		}
	}

	got := bl.query([]uint32{1, 5, 7, 10, 12}, nil)

	want := []uint16{'J' - 'A'}

	if !equalU16s(got, want) {
		t.Errorf("bl.query()=%v, want %v", got, want)
	}
}

func TestEndToEnd(t *testing.T) {

	idx := NewIndex(256, 1024, 4)

	docs := []string{
		`large black cat`,
		`the small grey dog`,
		`the large grey cat`,
		`small pumpkins`,
		`orange pumpkins`,
		`blue smurfs`,
		`small smurfs`,
		`small blue gophers`,
	}

	for _, d := range docs {

		tokens := strings.Fields(d)

		var toks []uint32

		for _, t := range tokens {
			toks = append(toks, crc32.ChecksumIEEE([]byte(t)))
		}

		idx.AddDocument(toks)
	}

	var toks []uint32

	query := []string{"smurfs"}

	for _, q := range query {
		toks = append(toks, crc32.ChecksumIEEE([]byte(q)))
	}

	ids := idx.Query(toks)

	want := []DocID{5, 6}

	if !equalU64s(ids, want) {
		t.Errorf("idx.Query(smurfs)=%v, want %v", ids, want)
	}
}

func TestShardEndToEnd(t *testing.T) {

	idx := NewShardedIndex(0.01, 4)

	docs := []string{
		`cat`,
		`dog`,
		`black cat`,
		`small dog`,
		`blue smurfs`,
		`small smurfs`,
		`small pumpkins`,
		`large black cat`,
		`orange pumpkins`,
		`the small grey dog`,
		`the large grey cat`,
		`small blue gophers`,
	}

	for _, d := range docs {
		tokens := strings.Fields(d)
		var toks []uint32
		for _, t := range tokens {
			toks = append(toks, crc32.ChecksumIEEE([]byte(t)))
		}
		idx.AddDocument(toks)
	}

	var toks []uint32

	query := []string{"smurfs"}

	for _, q := range query {
		toks = append(toks, crc32.ChecksumIEEE([]byte(q)))
	}

	ids := idx.Query(toks)

	want := []DocID{4, 5}

	if !equalU64s(ids, want) {
		t.Errorf("idx.Query(smurfs)=%v, want %v", ids, want)
	}
}

func TestPopset(t *testing.T) {

	var tests = []struct {
		u    bitrow
		want []uint16
	}{
		{bitrow{}, nil},
		{bitrow{1, 0}, []uint16{0}},
		{bitrow{3, 0}, []uint16{0, 1}},
		{bitrow{1<<12 | 1<<8 | 1<<4, 0}, []uint16{4, 8, 12}},
		{bitrow{0, 1<<12 | 1<<8 | 1<<4}, []uint16{4 + 64, 8 + 64, 12 + 64}},
	}

	for _, tt := range tests {
		got := popset(tt.u, nil)
		if !equalU16s(got, tt.want) {
			t.Errorf("popset(%d)=%v, want %v\n", tt.u, got, tt.want)
		}
	}
}

func equalU16s(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func equalU64s(a, b []DocID) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

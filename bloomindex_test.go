package bloomindex

import (
	"hash/crc32"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func TestBlockGetSet(t *testing.T) {

	bl := newBlock(64)
	bl.valid = 64

	set := make(map[uint16]bool)

	for i := 0; i < 1000; i++ {
		doc := uint16(rand.Intn(64))
		bit := uint16(rand.Intn(64))

		set[doc<<8+bit] = true

		bl.setbit(uint16(doc), bit)
	}

	for doc := uint16(0); doc < 64; doc++ {
		for bit := uint16(0); bit < 64; bit++ {
			want := set[doc<<8+bit]

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
			bl.setbit(d-'A', uint16(i))
		}
	}

	got := bl.query([]uint16{1, 5, 7, 10, 12})

	want := []uint16{'J' - 'A'}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("bl.query()=%v, want %v", got, want)
	}
}

func TestEndToEnd(t *testing.T) {

	idx := NewIndex(256, 4)

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

		docid := idx.AddDocument()
		tokens := strings.Fields(d)

		var toks []uint32

		for _, t := range tokens {
			toks = append(toks, crc32.ChecksumIEEE([]byte(t)))
		}

		idx.AddTerms(docid, toks)
	}

	var toks []uint32

	query := []string{"smurfs"}

	for _, q := range query {
		toks = append(toks, crc32.ChecksumIEEE([]byte(q)))
	}

	ids := idx.Query(toks)

	want := []DocID{5, 6}

	if !reflect.DeepEqual(ids, want) {
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
		got := popset(tt.u)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("popset(%d)=%v, want %v\n", tt.u, got, tt.want)
		}
	}
}

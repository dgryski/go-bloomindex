//go:build ignore
// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func main() {
	ConstraintExpr("amd64,!purego")

	TEXT("queryCore", NOSPLIT, "func(r *[8]uint64, bits [][8]uint64, hashes []uint32)")

	reg_r := GP64()
	reg_bits := GP64()
	reg_hashes := GP64()
	reg_length := GP64()

	Load(Param("r"), reg_r)
	Load(Param("bits").Base(), reg_bits)
	Load(Param("hashes").Base(), reg_hashes)
	Load(Param("hashes").Len(), reg_length)

	idx := GP64()

	var xmm_regs []Register

	for i := 0; i < 4; i++ {
		xmm_regs = append(xmm_regs, XMM())
	}

	xmm_tmp := XMM()

	// bits rows are only 8-byte aligned, so they must be loaded with MOVOU
	// rather than used directly as a PAND memory operand (which faults on
	// addresses that are not 16-byte aligned).
	var xmm_load []Register
	for range xmm_regs {
		xmm_load = append(xmm_load, XMM())
	}

	// generate -1 everywhere
	for _, v := range xmm_regs {
		PCMPEQL(v, v)
	}

	Label("loop")

	MOVL(Mem{Base: reg_hashes}, idx.As32())
	SHLQ(Imm(6), idx)
	ADDQ(reg_bits, idx)

	PXOR(xmm_tmp, xmm_tmp)

	for i, r := range xmm_regs {
		MOVOU(Mem{Base: idx, Disp: int(r.Size()) * i}, xmm_load[i])
	}

	for i, r := range xmm_regs {
		PAND(xmm_load[i], r)
	}

	for _, r := range xmm_regs {
		POR(r, xmm_tmp)
	}
	PTEST(xmm_tmp, xmm_tmp)
	JZ(LabelRef("done"))

	ADDQ(Imm(4), reg_hashes)
	SUBQ(Imm(1), reg_length)
	JNZ(LabelRef("loop"))

	Label("done")

	for i, r := range xmm_regs {
		MOVOU(r, Mem{Base: reg_r, Disp: int(r.Size()) * i})
	}

	RET()

	Generate()
}

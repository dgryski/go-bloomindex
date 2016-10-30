from peachpy import *
from peachpy.x86_64 import *

r = Argument(ptr(const_uint64_t))
bits = Argument(ptr(const_uint64_t))
bits_len = Argument(int64_t)
bits_cap = Argument(int64_t)
hashes = Argument(ptr(const_uint16_t))
hashes_len = Argument(int64_t)
hashes_cap = Argument(int64_t)

with Function("queryCore", (r, bits, bits_len, bits_cap, hashes, hashes_len, hashes_cap), target=uarch.default + isa.fma3) as function:
    reg_r = GeneralPurposeRegister64()
    reg_bits = GeneralPurposeRegister64()
    reg_hashes = GeneralPurposeRegister64()
    reg_length = GeneralPurposeRegister64()

    LOAD.ARGUMENT(reg_r, r)
    LOAD.ARGUMENT(reg_bits, bits)
    LOAD.ARGUMENT(reg_hashes, hashes)
    LOAD.ARGUMENT(reg_length, hashes_len)

    idx = GeneralPurposeRegister64()

    xmm_regs = [XMMRegister() for _ in range(4)]
    xmm_tmp = XMMRegister()

    # generate -1 everywhere
    for reg in xmm_regs:
        PCMPEQD(reg, reg)

    scalar_loop = Loop()
    with scalar_loop:
        XOR(idx, idx)
        MOV(idx.as_word, [reg_hashes])

        SHL(idx, 6)
        ADD(idx, reg_bits)
        PXOR(xmm_tmp, xmm_tmp)

        for i, reg in enumerate(xmm_regs):
            ANDPS(reg, [idx+16*i])

        for i, reg in enumerate(xmm_regs):
            POR(xmm_tmp, reg)
        PTEST(xmm_tmp, xmm_tmp)
        JZ(scalar_loop.end)

        ADD(reg_hashes, 2)

        SUB(reg_length, 1)
        JNZ(scalar_loop.begin)

    for i, reg in enumerate(xmm_regs):
        MOVAPS([reg_r+16*i], reg)

    RETURN()



//go:build amd64

#include "textflag.h"

// func indexByte2AVX2(b []byte, c1, c2 byte) int
TEXT ·indexByte2AVX2(SB), NOSPLIT, $0-40
	MOVQ b_base+0(FP), DI  // DI = b.ptr
	MOVQ b_len+8(FP), SI   // SI = b.len
	MOVBQZX c1+24(FP), AX  // AX = c1
	MOVBQZX c2+25(FP), CX  // CX = c2

	// Broadcast c1 and c2 into YMM0 and YMM1
	MOVQ AX, X0
	VPBROADCASTB X0, Y0
	MOVQ CX, X1
	VPBROADCASTB X1, Y1

	XORQ DX, DX            // DX = index

loop:
	MOVQ SI, R9
	SUBQ DX, R9            // R9 = remaining length (SI - DX)
	CMPQ R9, $32
	JL trailing            // if remaining < 32, go to trailing loop

	VMOVDQU (DI)(DX*1), Y2 // Load 32 bytes
	VPCMPEQB Y0, Y2, Y3    // Compare with c1
	VPCMPEQB Y1, Y2, Y4    // Compare with c2
	VPOR Y3, Y4, Y5        // bitwise OR the comparison results
	VPMOVMSKB Y5, R8       // Move 32-bit mask of matches to R8

	TESTL R8, R8
	JZ next_block          // if mask is 0, no matches in this block

	// Match found in this 32-byte block!
	BSFL R8, R8            // Find first set bit (index of match in block)
	ADDQ R8, DX            // DX = absolute index of match
	MOVQ DX, ret+32(FP)    // Return index
	VZEROUPPER
	RET

next_block:
	ADDQ $32, DX
	JMP loop

trailing:
	CMPQ DX, SI
	JE not_found           // if index == len, we are done

trail_loop:
	MOVB (DI)(DX*1), R8
	CMPB R8, AX
	JE found_trail
	CMPB R8, CX
	JE found_trail
	ADDQ $1, DX
	CMPQ DX, SI
	JL trail_loop

not_found:
	MOVQ $-1, ret+32(FP)
	VZEROUPPER
	RET

found_trail:
	MOVQ DX, ret+32(FP)
	VZEROUPPER
	RET

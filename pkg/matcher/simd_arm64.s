//go:build arm64

#include "textflag.h"

// func indexByte2NEON(b []byte, c1, c2 byte) int
//
// Scans b 16 bytes at a time using NEON/ASIMD (available on every arm64v8
// core, including Apple-silicon Macs). Each 16-byte block is compared against
// both target bytes; if any lane matches, a short scalar scan resolves the
// exact index. Any bytes that don't fill a full 16-byte vector are handled by
// the scalar trailing loop, so the vector loads never read past len(b).
//
// Caller (indexByte2SIMD) guarantees len(b) >= 16 before dispatching here, but
// the loop re-checks the remaining length every iteration so shorter tails are
// handled safely regardless.
TEXT ·indexByte2NEON(SB), NOSPLIT, $0-40
	MOVD	b_base+0(FP), R0   // R0 = b.ptr
	MOVD	b_len+8(FP), R1    // R1 = b.len
	MOVBU	c1+24(FP), R2      // R2 = c1
	MOVBU	c2+25(FP), R3      // R3 = c2

	VMOV	R2, V0.B16         // broadcast c1 into all 16 lanes
	VMOV	R3, V1.B16         // broadcast c2 into all 16 lanes

	MOVD	$0, R4             // R4 = index = 0

loop16:
	SUB	R4, R1, R6         // R6 = remaining = len - index
	CMP	$16, R6
	BLT	trailing           // remaining < 16 -> scalar tail

	ADD	R0, R4, R7         // R7 = &b[index]
	VLD1	(R7), [V2.B16]     // load 16 bytes
	VCMEQ	V0.B16, V2.B16, V3.B16 // lanes == c1 -> 0xFF
	VCMEQ	V1.B16, V2.B16, V4.B16 // lanes == c2 -> 0xFF
	VORR	V4.B16, V3.B16, V6.B16 // combine both comparisons

	// Reduce V6 to two GPR halves and OR them: non-zero iff any match.
	VMOV	V6.D[0], R8
	VMOV	V6.D[1], R9
	ORR	R9, R8, R8
	CBZ	R8, next16         // no match in this block

	// A match exists within the next 16 bytes; resolve its exact
	// position with the scalar loop (index is still < len here).
	B	trailing

next16:
	ADD	$16, R4, R4
	B	loop16

trailing:
	CMP	R1, R4
	BEQ	notfound           // index == len -> done

trail_loop:
	ADD	R0, R4, R7
	MOVBU	(R7), R8
	CMP	R2, R8
	BEQ	found
	CMP	R3, R8
	BEQ	found
	ADD	$1, R4, R4
	CMP	R1, R4
	BLT	trail_loop

notfound:
	MOVD	$-1, R0
	MOVD	R0, ret+32(FP)
	RET

found:
	MOVD	R4, ret+32(FP)
	RET

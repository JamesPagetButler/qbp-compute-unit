#include "textflag.h"

// mask1: [-, +, -, +]
DATA mask1<>+0(SB)/8, $0x8000000000000000
DATA mask1<>+8(SB)/8, $0x0000000000000000
DATA mask1<>+16(SB)/8, $0x8000000000000000
DATA mask1<>+24(SB)/8, $0x0000000000000000
GLOBL mask1<>(SB), RODATA, $32

// mask2: [-, +, +, -]
DATA mask2<>+0(SB)/8, $0x8000000000000000
DATA mask2<>+8(SB)/8, $0x0000000000000000
DATA mask2<>+16(SB)/8, $0x0000000000000000
DATA mask2<>+24(SB)/8, $0x8000000000000000
GLOBL mask2<>(SB), RODATA, $32

// mask3: [-, -, +, +]
DATA mask3<>+0(SB)/8, $0x8000000000000000
DATA mask3<>+8(SB)/8, $0x8000000000000000
DATA mask3<>+16(SB)/8, $0x0000000000000000
DATA mask3<>+24(SB)/8, $0x0000000000000000
GLOBL mask3<>(SB), RODATA, $32

// func mulAVX(q, r, res *Quat)
TEXT ·mulAVX(SB), NOSPLIT, $0-24
	MOVQ q+0(FP), AX
	MOVQ r+8(FP), BX
	MOVQ res+16(FP), CX

	// Load r into Y1
	VMOVUPD 0(BX), Y1

	// Broadcast q components
	VBROADCASTSD 0(AX), Y2  // q.W
	VBROADCASTSD 8(AX), Y3  // q.X
	VBROADCASTSD 16(AX), Y4 // q.Y
	VBROADCASTSD 24(AX), Y5 // q.Z

	// Term 0: q.W * r
	VMULPD Y1, Y2, Y6

	// Term 1: q.X * perm1(r) * mask1
	VSHUFPD $0x05, Y1, Y1, Y7
	VMOVUPD mask1<>(SB), Y10
	VXORPD Y10, Y7, Y7
	VMULPD Y7, Y3, Y3

	// Term 2: q.Y * perm2(r) * mask2
	VPERM2F128 $0x01, Y1, Y1, Y8
	VMOVUPD mask2<>(SB), Y11
	VXORPD Y11, Y8, Y14 // mutate into Y14 to preserve Y8
	VMULPD Y14, Y4, Y4

	// Term 3: q.Z * perm3(r) * mask3
	VSHUFPD $0x05, Y8, Y8, Y9 // safely uses untouched Y8
	VMOVUPD mask3<>(SB), Y12
	VXORPD Y12, Y9, Y9
	VMULPD Y9, Y5, Y5

	// Accumulate all 4 terms
	VADDPD Y6, Y3, Y13
	VADDPD Y13, Y4, Y13
	VADDPD Y13, Y5, Y13

	// Store result
	VMOVUPD Y13, 0(CX)
	VZEROUPPER
	RET

// func mulAccumAVX(dest, q, r *Quat)
TEXT ·mulAccumAVX(SB), NOSPLIT, $0-24
	MOVQ dest+0(FP), CX
	MOVQ q+8(FP), AX
	MOVQ r+16(FP), BX

	// Load r into Y1
	VMOVUPD 0(BX), Y1

	// Broadcast q components
	VBROADCASTSD 0(AX), Y2  // q.W
	VBROADCASTSD 8(AX), Y3  // q.X
	VBROADCASTSD 16(AX), Y4 // q.Y
	VBROADCASTSD 24(AX), Y5 // q.Z

	// Term 0: q.W * r
	VMULPD Y1, Y2, Y6

	// Term 1: q.X * perm1(r) * mask1
	VSHUFPD $0x05, Y1, Y1, Y7
	VMOVUPD mask1<>(SB), Y10
	VXORPD Y10, Y7, Y7
	VMULPD Y7, Y3, Y3

	// Term 2: q.Y * perm2(r) * mask2
	VPERM2F128 $0x01, Y1, Y1, Y8
	VMOVUPD mask2<>(SB), Y11
	VXORPD Y11, Y8, Y14 // mutate into Y14 to preserve Y8
	VMULPD Y14, Y4, Y4

	// Term 3: q.Z * perm3(r) * mask3
	VSHUFPD $0x05, Y8, Y8, Y9 // safely uses untouched Y8
	VMOVUPD mask3<>(SB), Y12
	VXORPD Y12, Y9, Y9
	VMULPD Y9, Y5, Y5

	// Accumulate all 4 terms
	VADDPD Y6, Y3, Y13
	VADDPD Y13, Y4, Y13
	VADDPD Y13, Y5, Y13

	// Add existing dest
	VMOVUPD 0(CX), Y15
	VADDPD Y13, Y15, Y13

	// Store result
	VMOVUPD Y13, 0(CX)
	VZEROUPPER
	RET

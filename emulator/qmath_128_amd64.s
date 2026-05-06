#include "textflag.h"

// Constants for sign flipping during conjugations
DATA y128_conj<>+0(SB)/8, $0x0000000000000000
DATA y128_conj<>+8(SB)/8, $0x8000000000000000
DATA y128_conj<>+16(SB)/8, $0x8000000000000000
DATA y128_conj<>+24(SB)/8, $0x8000000000000000
GLOBL y128_conj<>(SB), RODATA, $32

// func qconj128AVX(dst, a *QW128)
TEXT ·qconj128AVX(SB), NOSPLIT, $0-16
	MOVQ dst+0(FP), AX
	MOVQ a+8(FP), BX

	VMOVUPD 0(BX), Y0
	VMOVUPD 32(BX), Y1

	VMOVUPD y128_conj<>(SB), Y2
	VXORPD Y2, Y0, Y3
	VXORPD Y2, Y1, Y4

	VMOVUPD Y3, 0(AX)
	VMOVUPD Y4, 32(AX)
	VZEROUPPER
	RET

// func qadd128AVX(dst, a, b *QW128)
TEXT ·qadd128AVX(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), AX
	MOVQ a+8(FP), BX
	MOVQ b+16(FP), CX

	// Load A (Y0 = hi, Y1 = lo)
	VMOVUPD 0(BX), Y0
	VMOVUPD 32(BX), Y1

	// Load B (Y2 = hi, Y3 = lo)
	VMOVUPD 0(CX), Y2
	VMOVUPD 32(CX), Y3

	// a_hi + b_hi
	VADDPD Y2, Y0, Y4      // Y4 (s1) = a_hi + b_hi
	VSUBPD Y0, Y4, Y5      // Y5 (v) = s1 - a_hi
	VSUBPD Y5, Y4, Y6      // Y6 = s1 - v
	VSUBPD Y6, Y0, Y7      // Y7 = a_hi - Y6
	VSUBPD Y5, Y2, Y8      // Y8 = b_hi - v
	VADDPD Y8, Y7, Y9      // Y9 (e1) = Y7 + Y8

	// a_lo + b_lo + e1
	VADDPD Y3, Y1, Y10     // Y10 = a_lo + b_lo
	VADDPD Y9, Y10, Y11    // Y11 (s2) = Y10 + e1

	// s_hi, s_lo = fastTwoSum(s1, s2)
	VADDPD Y11, Y4, Y12    // Y12 (s_hi) = s1 + s2
	VSUBPD Y4, Y12, Y13    // Y13 = s_hi - s1
	VSUBPD Y13, Y11, Y14   // Y14 (s_lo) = s2 - Y13

	// Canonicalize: final_hi, final_lo = fastTwoSum(s_hi, s_lo)
	VADDPD Y14, Y12, Y15   // Y15 (final_hi) = s_hi + s_lo
	VSUBPD Y12, Y15, Y10   // Y10 = final_hi - s_hi
	VSUBPD Y10, Y14, Y11   // Y11 (final_lo) = s_lo - Y10

	// Store
	VMOVUPD Y15, 0(AX)
	VMOVUPD Y11, 32(AX)
	VZEROUPPER
	RET

DATA y128_sign_x<>+0(SB)/8, $0x8000000000000000 // W (-)
DATA y128_sign_x<>+8(SB)/8, $0x0000000000000000 // X (+)
DATA y128_sign_x<>+16(SB)/8, $0x8000000000000000 // Y (-)
DATA y128_sign_x<>+24(SB)/8, $0x0000000000000000 // Z (+)
GLOBL y128_sign_x<>(SB), RODATA, $32

DATA y128_sign_y<>+0(SB)/8, $0x8000000000000000 // W (-)
DATA y128_sign_y<>+8(SB)/8, $0x0000000000000000 // X (+)
DATA y128_sign_y<>+16(SB)/8, $0x0000000000000000 // Y (+)
DATA y128_sign_y<>+24(SB)/8, $0x8000000000000000 // Z (-)
GLOBL y128_sign_y<>(SB), RODATA, $32

DATA y128_sign_z<>+0(SB)/8, $0x8000000000000000 // W (-)
DATA y128_sign_z<>+8(SB)/8, $0x8000000000000000 // X (-)
DATA y128_sign_z<>+16(SB)/8, $0x0000000000000000 // Y (+)
DATA y128_sign_z<>+24(SB)/8, $0x0000000000000000 // Z (+)
GLOBL y128_sign_z<>(SB), RODATA, $32

// func qmul128AVX(dst, a, b *QW128)
TEXT ·qmul128AVX(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), AX
	MOVQ a+8(FP), BX
	MOVQ b+16(FP), CX

	// B_hi = Y2, B_lo = Y3
	VMOVUPD 0(CX), Y2
	VMOVUPD 32(CX), Y3

	// ------------------------------------------------------------------------
	// Term 0: a.W * b
	// ------------------------------------------------------------------------
	VMOVAPD Y2, Y6 // B_hi_shuf
	VMOVAPD Y3, Y7 // B_lo_shuf

	VBROADCASTSD 0(BX), Y8   // A.W_hi
	VBROADCASTSD 32(BX), Y9  // A.W_lo

	// ddMul
	VMULPD Y6, Y8, Y10
	VMOVAPD Y10, Y11
	VFMSUB231PD Y8, Y6, Y11
	VFMADD231PD Y7, Y8, Y11
	VFMADD231PD Y6, Y9, Y11
	VADDPD Y11, Y10, Y4      // p_hi -> Y4 (sum_hi)
	VSUBPD Y10, Y4, Y14
	VSUBPD Y14, Y11, Y5      // p_lo -> Y5 (sum_lo)

	// ------------------------------------------------------------------------
	// Term 1: a.X * b
	// ------------------------------------------------------------------------
	VSHUFPD $0x05, Y2, Y2, Y6 // B_hi_shuf
	VSHUFPD $0x05, Y3, Y3, Y7 // B_lo_shuf

	VBROADCASTSD 8(BX), Y8    // A.X_hi
	VBROADCASTSD 40(BX), Y9   // A.X_lo

	VMOVUPD y128_sign_x<>(SB), Y14
	VXORPD Y14, Y8, Y8
	VXORPD Y14, Y9, Y9

	// ddMul -> Y12, Y13
	VMULPD Y6, Y8, Y10
	VMOVAPD Y10, Y11
	VFMSUB231PD Y8, Y6, Y11
	VFMADD231PD Y7, Y8, Y11
	VFMADD231PD Y6, Y9, Y11
	VADDPD Y11, Y10, Y12
	VSUBPD Y10, Y12, Y14
	VSUBPD Y14, Y11, Y13

	// ddAdd to sum (Y4, Y5)
	VADDPD Y12, Y4, Y10
	VSUBPD Y4, Y10, Y11
	VSUBPD Y11, Y10, Y14
	VSUBPD Y14, Y4, Y15
	VSUBPD Y11, Y12, Y8
	VADDPD Y8, Y15, Y9

	VADDPD Y13, Y5, Y14
	VADDPD Y9, Y14, Y11

	VADDPD Y11, Y10, Y4
	VSUBPD Y10, Y4, Y14
	VSUBPD Y14, Y11, Y5

	// ------------------------------------------------------------------------
	// Term 2: a.Y * b
	// ------------------------------------------------------------------------
	VPERM2F128 $0x01, Y2, Y2, Y6
	VPERM2F128 $0x01, Y3, Y3, Y7

	VBROADCASTSD 16(BX), Y8   // A.Y_hi
	VBROADCASTSD 48(BX), Y9   // A.Y_lo

	VMOVUPD y128_sign_y<>(SB), Y14
	VXORPD Y14, Y8, Y8
	VXORPD Y14, Y9, Y9

	// ddMul -> Y12, Y13
	VMULPD Y6, Y8, Y10
	VMOVAPD Y10, Y11
	VFMSUB231PD Y8, Y6, Y11
	VFMADD231PD Y7, Y8, Y11
	VFMADD231PD Y6, Y9, Y11
	VADDPD Y11, Y10, Y12
	VSUBPD Y10, Y12, Y14
	VSUBPD Y14, Y11, Y13

	// ddAdd to sum (Y4, Y5)
	VADDPD Y12, Y4, Y10
	VSUBPD Y4, Y10, Y11
	VSUBPD Y11, Y10, Y14
	VSUBPD Y14, Y4, Y15
	VSUBPD Y11, Y12, Y8
	VADDPD Y8, Y15, Y9

	VADDPD Y13, Y5, Y14
	VADDPD Y9, Y14, Y11

	VADDPD Y11, Y10, Y4
	VSUBPD Y10, Y4, Y14
	VSUBPD Y14, Y11, Y5

	// ------------------------------------------------------------------------
	// Term 3: a.Z * b
	// ------------------------------------------------------------------------
	VPERM2F128 $0x01, Y2, Y2, Y6
	VSHUFPD $0x05, Y6, Y6, Y6
	VPERM2F128 $0x01, Y3, Y3, Y7
	VSHUFPD $0x05, Y7, Y7, Y7

	VBROADCASTSD 24(BX), Y8   // A.Z_hi
	VBROADCASTSD 56(BX), Y9   // A.Z_lo

	VMOVUPD y128_sign_z<>(SB), Y14
	VXORPD Y14, Y8, Y8
	VXORPD Y14, Y9, Y9

	// ddMul -> Y12, Y13
	VMULPD Y6, Y8, Y10
	VMOVAPD Y10, Y11
	VFMSUB231PD Y8, Y6, Y11
	VFMADD231PD Y7, Y8, Y11
	VFMADD231PD Y6, Y9, Y11
	VADDPD Y11, Y10, Y12
	VSUBPD Y10, Y12, Y14
	VSUBPD Y14, Y11, Y13

	// ddAdd to sum (Y4, Y5)
	VADDPD Y12, Y4, Y10
	VSUBPD Y4, Y10, Y11
	VSUBPD Y11, Y10, Y14
	VSUBPD Y14, Y4, Y15
	VSUBPD Y11, Y12, Y8
	VADDPD Y8, Y15, Y9

	VADDPD Y13, Y5, Y14
	VADDPD Y9, Y14, Y11

	VADDPD Y11, Y10, Y4
	VSUBPD Y10, Y4, Y14
	VSUBPD Y14, Y11, Y5

	// Canonicalize (Option B associative enforcement via twoSum)
	VADDPD Y5, Y4, Y15     // Y15 (final_hi, s) = Y4 + Y5
	VSUBPD Y4, Y15, Y10    // Y10 (v) = s - Y4
	VSUBPD Y10, Y15, Y11   // Y11 (s-v)
	VSUBPD Y11, Y4, Y12    // Y12 = Y4 - (s-v)
	VSUBPD Y10, Y5, Y13    // Y13 = Y5 - v
	VADDPD Y13, Y12, Y11   // Y11 (final_lo, e) = Y12 + Y13

	// Store
	VMOVUPD Y15, 0(AX)
	VMOVUPD Y11, 32(AX)
	VZEROUPPER
	RET

// func qrot128Stub(dst, q, v *QW128)
TEXT ·qrot128Stub(SB), NOSPLIT, $0-24
	JMP ·qrot128Scalar(SB)

// func qnorm128Stub(dst *QW128, a *QW128)
TEXT ·qnorm128Stub(SB), NOSPLIT, $0-16
	JMP ·qnorm128Scalar(SB)

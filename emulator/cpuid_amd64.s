#include "textflag.h"

// func hasAVXAndFMA() bool
TEXT ·hasAVXAndFMA(SB), NOSPLIT, $0-1
	MOVL $1, AX
	XORL CX, CX
	CPUID

	// Check ECX bits 27 (OSXSAVE), 28 (AVX), 12 (FMA)
	// 0x18001000
	MOVL $0x18001000, DX
	ANDL CX, DX
	CMPL DX, $0x18001000
	JNE no_avx

	// Check XGETBV
	XORL CX, CX // XCR0
	XGETBV
	// Check EAX bit 1 (SSE state) and bit 2 (AVX state) -> 0x6
	ANDL $0x6, AX
	CMPL AX, $0x6
	JNE no_avx

	MOVB $1, ret+0(FP)
	RET

no_avx:
	MOVB $0, ret+0(FP)
	RET

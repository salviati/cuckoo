// Copyright 2014 - Utkan Güngördü
// 
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

TEXT ·aeshash32(SB),4,$0-12
MOVL	k+0(FP), X0
MOVL	seed+4(FP), AX
PINSRD	$2, AX, X0
AESENC	·aeskeysched+0(SB), X0
AESENC	·aeskeysched+16(SB), X0
AESENC	·aeskeysched+0(SB), X0
MOVL	X0, ret+8(FP)
RET

// func aeshash32_4(k Key, mask Hash, seed *[4]Hash, h *[4]Hash)
TEXT ·aeshash32_4(SB),4,$0-24
MOVQ	seed+8(FP), SI
MOVQ	h+16(FP), DI

XORPS	X1, X1
MOVL	k+0(FP), X1
MOVAPS	X1, X4
PINSRD	$2, 0(SI), X4
MOVAPS	X1, X5
PINSRD	$2, 4(SI), X5
MOVAPS	X1, X6
PINSRD	$2, 8(SI), X6
MOVAPS	X1, X7
PINSRD	$2, 12(SI), X7

MOVUPS ·aeskeysched+0(SB), X0
MOVUPS ·aeskeysched+16(SB), X1


AESENC	X0, X4
AESENC	X1, X4
AESENC	X0, X4

AESENC	X0, X5
AESENC	X1, X5
AESENC	X0, X5

AESENC	X0, X6
AESENC	X1, X6
AESENC	X0, X6

AESENC	X0, X7
AESENC	X1, X7
AESENC	X0, X7

MOVL	X4, AX
PINSRD	$0, AX, X3

MOVL	X5, AX
PINSRD	$1, AX, X3

MOVL	X6, AX
PINSRD	$2, AX, X3

MOVL	X7, AX
PINSRD	$3, AX, X3

MOVL	mask+4(FP), X2 // mask
SHUFPS	$0, X2, X2
ANDPS	X2, X3
MOVUPS	X3, (DI)
RET

TEXT ·chkaesenc(SB),4,$0-0
MOVQ	$1, AX
CPUID
ANDL	$0x2000000, CX
JEQ	noaesenc
MOVB	$1, ·useaesenc+0(SB)
noaesenc:
RET

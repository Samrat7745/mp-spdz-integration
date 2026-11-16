// Copyright (c) 2021 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/mp-spdz-integration.
//
// SPDX-License-Identifier: Apache-2.0

// Package mpspdz enables conversion from big integers to the MP-SPDZ internal
// representation and vice versa.
package mpspdz

import (
	"errors"
	"fmt"
	"math/big"
)

const (
	// LimbWidth is the size of a limb in the MP-SPDZ runtime.
	LimbWidth = 8
	// WordWidth is the size of a word in the MP-SPDZ runtime.
	WordWidth = 2 * LimbWidth
	// ShareWidth is the size of a share (value, MAC) in the MP-SPDZ runtime.
	// Equals two times the word width.
	ShareWidth = 2 * WordWidth
)

// ErrNilParameter is returned by New if any of the parameters is nil.
var ErrNilParameter = errors.New("mpspdz: prime, r and rInv must not be nil")

// IntegrationUtils converts between big integers and the MP-SPDZ internal
// (Montgomery) representation. It is immutable and safe for concurrent use.
type IntegrationUtils struct {
	// prime is the modulus N as used by the MP-SPDZ implementation.
	prime *big.Int
	// r is the auxiliary modulus R as used by the MP-SPDZ implementation.
	r *big.Int
	// rInv is the multiplicative inverse of R as used by the MP-SPDZ implementation.
	rInv *big.Int
}

// New constructs an IntegrationUtils using prime (modulus N), r (auxiliary
// modulus R) and rInv (multiplicative inverse of R). The values are copied.
func New(prime, r, rInv *big.Int) (*IntegrationUtils, error) {
	if prime == nil || r == nil || rInv == nil {
		return nil, ErrNilParameter
	}
	return &IntegrationUtils{
		prime: new(big.Int).Set(prime),
		r:     new(big.Int).Set(r),
		rInv:  new(big.Int).Set(rInv),
	}, nil
}

// Prime returns a copy of the modulus N as used by the MP-SPDZ implementation.
func (u *IntegrationUtils) Prime() *big.Int { return new(big.Int).Set(u.prime) }

// R returns a copy of the auxiliary modulus R as used by the MP-SPDZ implementation.
func (u *IntegrationUtils) R() *big.Int { return new(big.Int).Set(u.r) }

// RInv returns a copy of the multiplicative inverse of R as used by the
// MP-SPDZ implementation.
func (u *IntegrationUtils) RInv() *big.Int { return new(big.Int).Set(u.rInv) }

// ToGfp converts value to the MP-SPDZ internal representation (WordWidth
// bytes). It returns an error if value is negative or larger than the
// configured prime.
func (u *IntegrationUtils) ToGfp(value *big.Int) ([]byte, error) {
	if value.Cmp(u.prime) > 0 {
		return nil, fmt.Errorf("value must not be larger than %s. Actual: %s", u.prime, value)
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("value must not be negative. Actual: %s", value)
	}
	mont := u.fromIntToMont(value)
	return swapLimbs(invertLimbEndianness(mont)), nil
}

// FromGfp converts a value in MP-SPDZ internal representation (WordWidth
// bytes) into a big integer. The result is never negative. It returns an
// error if gfp is not exactly WordWidth bytes long.
func (u *IntegrationUtils) FromGfp(gfp []byte) (*big.Int, error) {
	if len(gfp) != WordWidth {
		return nil, fmt.Errorf("gfp byte representation must have a length of %d. Actual: %d", WordWidth, len(gfp))
	}
	return u.fromMontToInt(swapLimbs(invertLimbEndianness(gfp))), nil
}

// invertLimbEndianness returns a new slice in which the byte order of each
// 8-byte limb of the word is reversed (little-endian <-> big-endian). MP-SPDZ
// stores values as two little-endian 8-byte limbs.
func invertLimbEndianness(input []byte) []byte {
	fixed := make([]byte, WordWidth)
	for i := 0; i < WordWidth/LimbWidth; i++ {
		off := i * LimbWidth
		for j := 0; j < LimbWidth; j++ {
			fixed[off+j] = input[off+LimbWidth-1-j]
		}
	}
	return fixed
}

// swapLimbs returns a new slice with the two 8-byte limbs swapped, i.e.
// limb[0], limb[1] becomes limb[1], limb[0].
func swapLimbs(input []byte) []byte {
	swapped := make([]byte, WordWidth)
	copy(swapped[LimbWidth:], input[:LimbWidth])
	copy(swapped[:LimbWidth], input[LimbWidth:WordWidth])
	return swapped
}

// fromMontToInt converts a big-endian unsigned integer in Montgomery form
// (with respect to prime and R) to its plain value modulo prime.
func (u *IntegrationUtils) fromMontToInt(value []byte) *big.Int {
	x := new(big.Int).SetBytes(value)
	x.Mul(x, u.rInv)
	return x.Mod(x, u.prime)
}

// fromIntToMont converts num (modulo prime) to its Montgomery form with
// respect to prime and R, as a big-endian WordWidth-byte slice. Values wider
// than WordWidth bytes are truncated to their least significant bytes.
func (u *IntegrationUtils) fromIntToMont(num *big.Int) []byte {
	m := new(big.Int).Mul(num, u.r)
	m.Mod(m, u.prime)
	b := m.Bytes()
	if len(b) > WordWidth {
		b = b[len(b)-WordWidth:]
	}
	out := make([]byte, WordWidth)
	copy(out[WordWidth-len(b):], b)
	return out
}

// Copyright (c) 2021 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/mp-spdz-integration.
//
// SPDX-License-Identifier: Apache-2.0

package mpspdz

import (
	"bufio"
	"bytes"
	"errors"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"
)

const repetitions = 100000

var (
	testPrime = mustBigInt("198766463529478683931867765928436695041")
	testR     = mustBigInt("141515903391459779531506841503331516415")
	testRInv  = mustBigInt("133854242216446749056083838363708373830")
)

func mustBigInt(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big integer literal: " + s)
	}
	return v
}

type testPair struct {
	gfp   []byte
	value *big.Int
}

// loadTestData reads the MP-SPDZ gfp representations and the related human
// readable values (reduced modulo prime) from testdata.
func loadTestData(t *testing.T, prime *big.Int) []testPair {
	t.Helper()
	raw, err := os.ReadFile("testdata/GfpTestData")
	if err != nil {
		t.Fatalf("cannot open GFp test data: %v", err)
	}
	if len(raw)%WordWidth != 0 {
		t.Fatalf("GFp test data length %d is not a multiple of %d", len(raw), WordWidth)
	}
	f, err := os.Open("testdata/BigIntTestData")
	if err != nil {
		t.Fatalf("cannot open human readable test data: %v", err)
	}
	defer f.Close()

	var values []*big.Int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		values = append(values, new(big.Int).Mod(mustBigInt(strings.TrimSpace(sc.Text())), prime))
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("reading human readable test data: %v", err)
	}
	if n := len(raw) / WordWidth; n != len(values) {
		t.Fatalf("number of GFp values (%d) does not match number of human readable values (%d)", n, len(values))
	}
	pairs := make([]testPair, len(values))
	for i, v := range values {
		pairs[i] = testPair{gfp: raw[i*WordWidth : (i+1)*WordWidth], value: v}
	}
	return pairs
}

func newTestUtils(t *testing.T) *IntegrationUtils {
	t.Helper()
	u, err := New(testPrime, testR, testRInv)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return u
}

func TestNewWithNilValuesReturnsError(t *testing.T) {
	cases := [][3]*big.Int{
		{nil, testR, testRInv},
		{testPrime, nil, testRInv},
		{testPrime, testR, nil},
	}
	for _, c := range cases {
		if _, err := New(c[0], c[1], c[2]); !errors.Is(err, ErrNilParameter) {
			t.Errorf("New(%v, %v, %v): expected ErrNilParameter, got %v", c[0], c[1], c[2], err)
		}
	}
}

func TestGettersReturnConfiguredValues(t *testing.T) {
	u := newTestUtils(t)
	if u.Prime().Cmp(testPrime) != 0 || u.R().Cmp(testR) != 0 || u.RInv().Cmp(testRInv) != 0 {
		t.Fatal("getters do not return configured values")
	}
	// Mutating returned values must not affect the instance.
	u.Prime().SetInt64(0)
	if u.Prime().Cmp(testPrime) != 0 {
		t.Fatal("Prime() exposes internal state")
	}
}

func TestFromGfpReturnsCorrectOutput(t *testing.T) {
	u := newTestUtils(t)
	for i, p := range loadTestData(t, u.Prime()) {
		got, err := u.FromGfp(p.gfp)
		if err != nil {
			t.Fatalf("#%d: unexpected error: %v", i, err)
		}
		if got.Cmp(p.value) != 0 {
			t.Fatalf("#%d: converted value does not match actual value: got %s, want %s", i, got, p.value)
		}
	}
}

func TestFromGfpWithWrongLengthReturnsError(t *testing.T) {
	_, err := newTestUtils(t).FromGfp([]byte{4, 2})
	if err == nil || !strings.Contains(err.Error(), "must have a length of") {
		t.Fatalf("expected length error, got %v", err)
	}
}

func TestToGfpReturnsCorrectOutput(t *testing.T) {
	u := newTestUtils(t)
	for i, p := range loadTestData(t, u.Prime()) {
		got, err := u.ToGfp(p.value)
		if err != nil {
			t.Fatalf("#%d: unexpected error: %v", i, err)
		}
		if !bytes.Equal(got, p.gfp) {
			t.Fatalf("#%d: converted byte data does not match actual spdz representation: got %x, want %x", i, got, p.gfp)
		}
	}
}

func TestToGfpWithNegativeValueReturnsError(t *testing.T) {
	_, err := newTestUtils(t).ToGfp(big.NewInt(-1))
	if err == nil || !strings.Contains(err.Error(), "must not be negative") {
		t.Fatalf("expected negative value error, got %v", err)
	}
}

func TestToGfpWithValueGreaterThanPrimeReturnsError(t *testing.T) {
	u := newTestUtils(t)
	_, err := u.ToGfp(new(big.Int).Add(u.Prime(), big.NewInt(1)))
	if err == nil || !strings.Contains(err.Error(), "must not be larger") {
		t.Fatalf("expected too large error, got %v", err)
	}
}

func TestRandomRoundtripPreservesValue(t *testing.T) {
	u := newTestUtils(t)
	rnd := rand.New(rand.NewSource(42))
	for i := 0; i < repetitions; i++ {
		v := rnd.Int63()
		gfp, err := u.ToGfp(big.NewInt(v))
		if err != nil {
			t.Fatalf("ToGfp(%d): %v", v, err)
		}
		got, err := u.FromGfp(gfp)
		if err != nil {
			t.Fatalf("FromGfp: %v", err)
		}
		if !got.IsInt64() || got.Int64() != v {
			t.Fatalf("roundtrip does not preserve value: got %s, want %d", got, v)
		}
	}
}

// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package bitutil

import (
	"bytes"
	"testing"
)

// Tests that bitwise XOR works for various alignments.
func TestSilaXOR(t *testing.T) {
	for alignP := 0; alignP < 2; alignP++ {
		for alignQ := 0; alignQ < 2; alignQ++ {
			for alignD := 0; alignD < 2; alignD++ {
				p := make([]byte, 1023)[alignP:]
				q := make([]byte, 1023)[alignQ:]

				for i := 0; i < len(p); i++ {
					p[i] = byte(i)
				}
				for i := 0; i < len(q); i++ {
					q[i] = byte(len(q) - i)
				}

				d1 := make([]byte, 1023+alignD)[alignD:]
				d2 := make([]byte, 1023+alignD)[alignD:]

				SilaXORBytes(d1, p, q)
				silaNaiveXOR(d2, p, q)

				if !bytes.Equal(d1, d2) {
					t.Error("not equal", d1, d2)
				}
			}
		}
	}
}

// silaNaiveXOR xors bytes one by one.
func silaNaiveXOR(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

// Tests that bitwise AND works for various alignments.
func TestSilaAND(t *testing.T) {
	for alignP := 0; alignP < 2; alignP++ {
		for alignQ := 0; alignQ < 2; alignQ++ {
			for alignD := 0; alignD < 2; alignD++ {
				p := make([]byte, 1023)[alignP:]
				q := make([]byte, 1023)[alignQ:]

				for i := 0; i < len(p); i++ {
					p[i] = byte(i)
				}
				for i := 0; i < len(q); i++ {
					q[i] = byte(len(q) - i)
				}

				d1 := make([]byte, 1023+alignD)[alignD:]
				d2 := make([]byte, 1023+alignD)[alignD:]

				SilaANDBytes(d1, p, q)
				silaSafeANDBytes(d2, p, q)

				if !bytes.Equal(d1, d2) {
					t.Error("not equal")
				}
			}
		}
	}
}

// Tests that bitwise OR works for various alignments.
func TestSilaOR(t *testing.T) {
	for alignP := 0; alignP < 2; alignP++ {
		for alignQ := 0; alignQ < 2; alignQ++ {
			for alignD := 0; alignD < 2; alignD++ {
				p := make([]byte, 1023)[alignP:]
				q := make([]byte, 1023)[alignQ:]

				for i := 0; i < len(p); i++ {
					p[i] = byte(i)
				}
				for i := 0; i < len(q); i++ {
					q[i] = byte(len(q) - i)
				}

				d1 := make([]byte, 1023+alignD)[alignD:]
				d2 := make([]byte, 1023+alignD)[alignD:]

				SilaORBytes(d1, p, q)
				silaSafeORBytes(d2, p, q)

				if !bytes.Equal(d1, d2) {
					t.Error("not equal")
				}
			}
		}
	}
}

// Tests that bit testing works for various alignments.
func TestSilaTestBytes(t *testing.T) {
	for align := 0; align < 2; align++ {
		p := make([]byte, 1023)[align:]
		p[100] = 1

		if SilaTestBytes(p) != silaSafeTestBytes(p) {
			t.Error("not equal")
		}

		q := make([]byte, 1023)[align:]
		q[len(q)-1] = 1

		if SilaTestBytes(q) != silaSafeTestBytes(q) {
			t.Error("not equal")
		}
	}
}

// Benchmarks the optimized XOR performance.
func BenchmarkSilaFastXOR1KB(b *testing.B) { silaBenchmarkFastXOR(b, 1024) }
func BenchmarkSilaFastXOR2KB(b *testing.B) { silaBenchmarkFastXOR(b, 2048) }
func BenchmarkSilaFastXOR4KB(b *testing.B) { silaBenchmarkFastXOR(b, 4096) }

func silaBenchmarkFastXOR(b *testing.B, size int) {
	p, q := make([]byte, size), make([]byte, size)
	for i := 0; i < b.N; i++ {
		SilaXORBytes(p, p, q)
	}
}

// Benchmarks the baseline XOR performance.
func BenchmarkSilaBaseXOR1KB(b *testing.B) { silaBenchmarkBaseXOR(b, 1024) }
func BenchmarkSilaBaseXOR2KB(b *testing.B) { silaBenchmarkBaseXOR(b, 2048) }
func BenchmarkSilaBaseXOR4KB(b *testing.B) { silaBenchmarkBaseXOR(b, 4096) }

func silaBenchmarkBaseXOR(b *testing.B, size int) {
	p, q := make([]byte, size), make([]byte, size)
	for i := 0; i < b.N; i++ {
		silaNaiveXOR(p, p, q)
	}
}

// Benchmarks the optimized AND performance.
func BenchmarkSilaFastAND1KB(b *testing.B) { silaBenchmarkFastAND(b, 1024) }
func BenchmarkSilaFastAND2KB(b *testing.B) { silaBenchmarkFastAND(b, 2048) }
func BenchmarkSilaFastAND4KB(b *testing.B) { silaBenchmarkFastAND(b, 4096) }

func silaBenchmarkFastAND(b *testing.B, size int) {
	p, q := make([]byte, size), make([]byte, size)
	for i := 0; i < b.N; i++ {
		SilaANDBytes(p, p, q)
	}
}

// Benchmarks the baseline AND performance.
func BenchmarkSilaBaseAND1KB(b *testing.B) { silaBenchmarkBaseAND(b, 1024) }
func BenchmarkSilaBaseAND2KB(b *testing.B) { silaBenchmarkBaseAND(b, 2048) }
func BenchmarkSilaBaseAND4KB(b *testing.B) { silaBenchmarkBaseAND(b, 4096) }

func silaBenchmarkBaseAND(b *testing.B, size int) {
	p, q := make([]byte, size), make([]byte, size)
	for i := 0; i < b.N; i++ {
		silaSafeANDBytes(p, p, q)
	}
}

// Benchmarks the optimized OR performance.
func BenchmarkSilaFastOR1KB(b *testing.B) { silaBenchmarkFastOR(b, 1024) }
func BenchmarkSilaFastOR2KB(b *testing.B) { silaBenchmarkFastOR(b, 2048) }
func BenchmarkSilaFastOR4KB(b *testing.B) { silaBenchmarkFastOR(b, 4096) }

func silaBenchmarkFastOR(b *testing.B, size int) {
	p, q := make([]byte, size), make([]byte, size)
	for i := 0; i < b.N; i++ {
		SilaORBytes(p, p, q)
	}
}

// Benchmarks the baseline OR performance.
func BenchmarkSilaBaseOR1KB(b *testing.B) { silaBenchmarkBaseOR(b, 1024) }
func BenchmarkSilaBaseOR2KB(b *testing.B) { silaBenchmarkBaseOR(b, 2048) }
func BenchmarkSilaBaseOR4KB(b *testing.B) { silaBenchmarkBaseOR(b, 4096) }

func silaBenchmarkBaseOR(b *testing.B, size int) {
	p, q := make([]byte, size), make([]byte, size)
	for i := 0; i < b.N; i++ {
		silaSafeORBytes(p, p, q)
	}
}

var SilaGlobalBool bool

// Benchmarks the optimized bit testing performance.
func BenchmarkSilaFastTest1KB(b *testing.B) { silaBenchmarkFastTest(b, 1024) }
func BenchmarkSilaFastTest2KB(b *testing.B) { silaBenchmarkFastTest(b, 2048) }
func BenchmarkSilaFastTest4KB(b *testing.B) { silaBenchmarkFastTest(b, 4096) }

func silaBenchmarkFastTest(b *testing.B, size int) {
	p := make([]byte, size)
	a := false
	for i := 0; i < b.N; i++ {
		a = a != SilaTestBytes(p)
	}
	SilaGlobalBool = a
}

// Benchmarks the baseline bit testing performance.
func BenchmarkSilaBaseTest1KB(b *testing.B) { silaBenchmarkBaseTest(b, 1024) }
func BenchmarkSilaBaseTest2KB(b *testing.B) { silaBenchmarkBaseTest(b, 2048) }
func BenchmarkSilaBaseTest4KB(b *testing.B) { silaBenchmarkBaseTest(b, 4096) }

func silaBenchmarkBaseTest(b *testing.B, size int) {
	p := make([]byte, size)
	a := false
	for i := 0; i < b.N; i++ {
		a = a != silaSafeTestBytes(p)
	}
	SilaGlobalBool = a
}

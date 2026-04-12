// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package bitutil

import (
	"crypto/subtle"
	"runtime"
	"unsafe"
)

const silaWordSize = int(unsafe.Sizeof(uintptr(0)))

const silaSupportsUnaligned = runtime.GOARCH == "386" ||
	runtime.GOARCH == "amd64" ||
	runtime.GOARCH == "ppc64" ||
	runtime.GOARCH == "ppc64le" ||
	runtime.GOARCH == "s390x"

// SilaXORBytes xors the bytes in a and b into dst and returns the number of processed bytes.
func SilaXORBytes(dst, a, b []byte) int {
	return subtle.XORBytes(dst, a, b)
}

// SilaANDBytes ands the bytes in a and b into dst and returns the number of processed bytes.
func SilaANDBytes(dst, a, b []byte) int {
	if silaSupportsUnaligned {
		return silaFastANDBytes(dst, a, b)
	}
	return silaSafeANDBytes(dst, a, b)
}

func silaFastANDBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	w := n / silaWordSize
	if w > 0 {
		dw := *(*[]uintptr)(unsafe.Pointer(&dst))
		aw := *(*[]uintptr)(unsafe.Pointer(&a))
		bw := *(*[]uintptr)(unsafe.Pointer(&b))
		for i := 0; i < w; i++ {
			dw[i] = aw[i] & bw[i]
		}
	}

	for i := n - n%silaWordSize; i < n; i++ {
		dst[i] = a[i] & b[i]
	}
	return n
}

func silaSafeANDBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	for i := 0; i < n; i++ {
		dst[i] = a[i] & b[i]
	}
	return n
}

// SilaORBytes ors the bytes in a and b into dst and returns the number of processed bytes.
func SilaORBytes(dst, a, b []byte) int {
	if silaSupportsUnaligned {
		return silaFastORBytes(dst, a, b)
	}
	return silaSafeORBytes(dst, a, b)
}

func silaFastORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	w := n / silaWordSize
	if w > 0 {
		dw := *(*[]uintptr)(unsafe.Pointer(&dst))
		aw := *(*[]uintptr)(unsafe.Pointer(&a))
		bw := *(*[]uintptr)(unsafe.Pointer(&b))
		for i := 0; i < w; i++ {
			dw[i] = aw[i] | bw[i]
		}
	}

	for i := n - n%silaWordSize; i < n; i++ {
		dst[i] = a[i] | b[i]
	}
	return n
}

func silaSafeORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	for i := 0; i < n; i++ {
		dst[i] = a[i] | b[i]
	}
	return n
}

// SilaTestBytes reports whether any bit is set in p.
func SilaTestBytes(p []byte) bool {
	if silaSupportsUnaligned {
		return silaFastTestBytes(p)
	}
	return silaSafeTestBytes(p)
}

func silaFastTestBytes(p []byte) bool {
	n := len(p)
	w := n / silaWordSize

	if w > 0 {
		pw := *(*[]uintptr)(unsafe.Pointer(&p))
		for i := 0; i < w; i++ {
			if pw[i] != 0 {
				return true
			}
		}
	}

	for i := n - n%silaWordSize; i < n; i++ {
		if p[i] != 0 {
			return true
		}
	}

	return false
}

func silaSafeTestBytes(p []byte) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return true
		}
	}
	return false
}

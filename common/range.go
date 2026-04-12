// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"iter"
)

// SilaRange represents a range of unsigned integers.
type SilaRange[T uint32 | uint64] struct {
	first, afterLast T
}

// NewSilaRange creates a new range based on first element and number of elements.
func NewSilaRange[T uint32 | uint64](first, count T) SilaRange[T] {
	return SilaRange[T]{first: first, afterLast: first + count}
}

// First returns the first element of the range.
func (r SilaRange[T]) First() T {
	return r.first
}

// Last returns the last element of the range. This panics for empty ranges.
func (r SilaRange[T]) Last() T {
	if r.first == r.afterLast {
		panic("last item of zero length range is not allowed")
	}
	return r.afterLast - 1
}

// AfterLast returns the first element after the range.
func (r SilaRange[T]) AfterLast() T {
	return r.afterLast
}

// Count returns the number of elements in the range.
func (r SilaRange[T]) Count() T {
	return r.afterLast - r.first
}

// IsEmpty returns true if the range is empty.
func (r SilaRange[T]) IsEmpty() bool {
	return r.first == r.afterLast
}

// Includes returns true if the given element is inside the range.
func (r SilaRange[T]) Includes(v T) bool {
	return v >= r.first && v < r.afterLast
}

// SetFirst updates the first element of the range.
func (r *SilaRange[T]) SetFirst(v T) {
	r.first = v
	if r.afterLast < r.first {
		r.afterLast = r.first
	}
}

// SetAfterLast updates the end of the range by specifying the first element after the range.
func (r *SilaRange[T]) SetAfterLast(v T) {
	r.afterLast = v
	if r.afterLast < r.first {
		r.first = r.afterLast
	}
}

// SetLast updates the last element of the range.
func (r *SilaRange[T]) SetLast(v T) {
	r.SetAfterLast(v + 1)
}

// Intersection returns the intersection of two ranges.
func (r SilaRange[T]) Intersection(q SilaRange[T]) SilaRange[T] {
	i := SilaRange[T]{
		first:     silaMax(r.first, q.first),
		afterLast: silaMin(r.afterLast, q.afterLast),
	}
	if i.first > i.afterLast {
		return SilaRange[T]{}
	}
	return i
}

// Union returns the union of two ranges. Panics for gapped ranges.
func (r SilaRange[T]) Union(q SilaRange[T]) SilaRange[T] {
	if silaMax(r.first, q.first) > silaMin(r.afterLast, q.afterLast) {
		panic("cannot create union; gap between ranges")
	}
	return SilaRange[T]{
		first:     silaMin(r.first, q.first),
		afterLast: silaMax(r.afterLast, q.afterLast),
	}
}

// Iter iterates all integers in the range.
func (r SilaRange[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := r.first; i < r.afterLast; i++ {
			if !yield(i) {
				break
			}
		}
	}
}

func silaMin[T uint32 | uint64](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func silaMax[T uint32 | uint64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Copyright 2026 The SILA Authors
// This file is part of the sila-library.
//
// The sila-library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila-library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila-library. If not, see <http://www.gnu.org/licenses/>.

package state

// SILAContractCodeReaderStats aggregates statistics for the SILA contract code reader.
type SILAContractCodeReaderStats struct {
	CacheHit       int64 // Number of cache hits
	CacheMiss      int64 // Number of cache misses
	CacheHitBytes  int64 // Total bytes served from cache
	CacheMissBytes int64 // Total bytes read on cache misses
}

// HitRate returns the SILA cache hit rate in percentage.
func (s SILAContractCodeReaderStats) HitRate() float64 {
	total := s.CacheHit + s.CacheMiss
	if total == 0 {
		return 0
	}
	return float64(s.CacheHit) / float64(total) * 100
}

// SILAContractCodeReaderStater wraps the method to retrieve the statistics of
// SILA contract code reader.
type SILAContractCodeReaderStater interface {
	GetSILACodeStats() SILAContractCodeReaderStats
}

// SILAStateReaderStats aggregates statistics for the SILA state reader.
type SILAStateReaderStats struct {
	AccountCacheHit  int64 // Number of account cache hits
	AccountCacheMiss int64 // Number of account cache misses
	StorageCacheHit  int64 // Number of storage cache hits
	StorageCacheMiss int64 // Number of storage cache misses
}

// AccountCacheHitRate returns the SILA cache hit rate of account requests in percentage.
func (s SILAStateReaderStats) AccountCacheHitRate() float64 {
	total := s.AccountCacheHit + s.AccountCacheMiss
	if total == 0 {
		return 0
	}
	return float64(s.AccountCacheHit) / float64(total) * 100
}

// StorageCacheHitRate returns the SILA cache hit rate of storage requests in percentage.
func (s SILAStateReaderStats) StorageCacheHitRate() float64 {
	total := s.StorageCacheHit + s.StorageCacheMiss
	if total == 0 {
		return 0
	}
	return float64(s.StorageCacheHit) / float64(total) * 100
}

// SILAStateReaderStater wraps the method to retrieve the statistics of SILA state reader.
type SILAStateReaderStater interface {
	GetSILAStateStats() SILAStateReaderStats
}

// SILAReaderStats wraps the statistics of SILA reader.
type SILAReaderStats struct {
	CodeStats  SILAContractCodeReaderStats
	StateStats SILAStateReaderStats
}

// SILAReaderStater defines the capability to retrieve aggregated statistics.
type SILAReaderStater interface {
	GetSILAStats() SILAReaderStats
}

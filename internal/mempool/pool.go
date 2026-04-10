package mempool

import (
	"errors"
	"sync"

	"silachain/internal/core/types"
)

const (
	DefaultMaxPoolSize        = 1024
	DefaultMaxPerSender       = 16
	DefaultMaxFutureNonceGap  = 8
	DefaultReplacementBumpBps = 1000 // 10.00%
)

var (
	ErrNilPool                = errors.New("mempool is nil")
	ErrNilTx                  = errors.New("transaction is nil")
	ErrDuplicateTx            = errors.New("duplicate transaction")
	ErrDuplicateSenderNonce   = errors.New("duplicate sender nonce")
	ErrReplacementUnderpriced = errors.New("replacement transaction underpriced")
	ErrPoolFull               = errors.New("mempool is full")
	ErrSenderQueueFull        = errors.New("sender mempool quota exceeded")
	ErrNonceTooFarInFuture    = errors.New("nonce too far in future")
)

type senderNonceKey struct {
	From  types.Address
	Nonce types.Nonce
}

type Pool struct {
	mu                 sync.RWMutex
	txs                map[types.Hash]*types.Transaction
	senderNonceIndex   map[senderNonceKey]types.Hash
	senderCounts       map[types.Address]int
	maxPoolSize        int
	maxPerSender       int
	maxFutureNonceGap  types.Nonce
	replacementBumpBps uint64
}

func NewPool() *Pool {
	return &Pool{
		txs:                make(map[types.Hash]*types.Transaction),
		senderNonceIndex:   make(map[senderNonceKey]types.Hash),
		senderCounts:       make(map[types.Address]int),
		maxPoolSize:        DefaultMaxPoolSize,
		maxPerSender:       DefaultMaxPerSender,
		maxFutureNonceGap:  DefaultMaxFutureNonceGap,
		replacementBumpBps: DefaultReplacementBumpBps,
	}
}

func (p *Pool) SetLimits(maxPoolSize int, maxPerSender int, maxFutureNonceGap types.Nonce) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if maxPoolSize > 0 {
		p.maxPoolSize = maxPoolSize
	}
	if maxPerSender > 0 {
		p.maxPerSender = maxPerSender
	}
	if maxFutureNonceGap > 0 {
		p.maxFutureNonceGap = maxFutureNonceGap
	}
}

func (p *Pool) SetReplacementBumpBps(bps uint64) {
	if p == nil || bps == 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.replacementBumpBps = bps
}

func (p *Pool) Add(t *types.Transaction) error {
	if p == nil {
		return ErrNilPool
	}
	if t == nil {
		return ErrNilTx
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.txs[t.Hash]; exists {
		return ErrDuplicateTx
	}

	key := senderNonceKey{From: t.From, Nonce: t.Nonce}
	if existingHash, exists := p.senderNonceIndex[key]; exists {
		existingTx, ok := p.txs[existingHash]
		if !ok || existingTx == nil {
			delete(p.senderNonceIndex, key)
		} else {
			if !replacementAllowed(t, existingTx, p.replacementBumpBps) {
				return ErrReplacementUnderpriced
			}
			p.removeTxLocked(existingHash, existingTx)
		}
	}

	if p.maxPerSender > 0 && p.senderCounts[t.From] >= p.maxPerSender {
		return ErrSenderQueueFull
	}

	if p.maxPoolSize > 0 && len(p.txs) >= p.maxPoolSize {
		evictHash, evictTx, ok := p.lowestEvictableLocked()
		if !ok {
			return ErrPoolFull
		}
		if !betterPriority(t, evictTx) {
			return ErrPoolFull
		}
		p.removeTxLocked(evictHash, evictTx)
	}

	p.txs[t.Hash] = t
	p.senderNonceIndex[key] = t.Hash
	p.senderCounts[t.From]++

	return nil
}

func (p *Pool) Count() int {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.txs)
}

func (p *Pool) Pending() []types.Transaction {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]types.Transaction, 0, len(p.txs))
	for _, t := range p.txs {
		out = append(out, *t)
	}

	return out
}

func (p *Pool) HasHash(hash types.Hash) bool {
	if p == nil {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	_, ok := p.txs[hash]
	return ok
}

func (p *Pool) HasSenderNonce(from types.Address, nonce types.Nonce) bool {
	if p == nil {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	_, ok := p.senderNonceIndex[senderNonceKey{From: from, Nonce: nonce}]
	return ok
}

func (p *Pool) SenderCount(from types.Address) int {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.senderCounts[from]
}

func (p *Pool) Clear() {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.txs = make(map[types.Hash]*types.Transaction)
	p.senderNonceIndex = make(map[senderNonceKey]types.Hash)
	p.senderCounts = make(map[types.Address]int)
}

func (p *Pool) NextNonceForSender(from types.Address, currentNonce types.Nonce) types.Nonce {
	if p == nil {
		return currentNonce
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	next := currentNonce
	for {
		_, ok := p.senderNonceIndex[senderNonceKey{From: from, Nonce: next}]
		if !ok {
			return next
		}
		next++
	}
}

func (p *Pool) removeTxLocked(hash types.Hash, t *types.Transaction) {
	if p == nil || t == nil {
		return
	}

	delete(p.txs, hash)
	delete(p.senderNonceIndex, senderNonceKey{From: t.From, Nonce: t.Nonce})

	if p.senderCounts[t.From] <= 1 {
		delete(p.senderCounts, t.From)
	} else {
		p.senderCounts[t.From]--
	}
}

func (p *Pool) isTailNonceLocked(from types.Address, nonce types.Nonce) bool {
	for key := range p.senderNonceIndex {
		if key.From == from && key.Nonce > nonce {
			return false
		}
	}
	return true
}

func (p *Pool) lowestEvictableLocked() (types.Hash, *types.Transaction, bool) {
	var selectedHash types.Hash
	var selectedTx *types.Transaction
	found := false

	for hash, candidate := range p.txs {
		if candidate == nil {
			continue
		}
		if !p.isTailNonceLocked(candidate.From, candidate.Nonce) {
			continue
		}
		if !found || worsePriority(candidate, selectedTx) {
			selectedHash = hash
			selectedTx = candidate
			found = true
		}
	}

	return selectedHash, selectedTx, found
}

func replacementAllowed(newTx *types.Transaction, oldTx *types.Transaction, bumpBps uint64) bool {
	if newTx == nil || oldTx == nil {
		return false
	}

	newFee := uint64(newTx.EffectiveFee())
	oldFee := uint64(oldTx.EffectiveFee())

	if newFee <= oldFee {
		return false
	}

	required := oldFee
	if bumpBps > 0 {
		required = oldFee + ((oldFee * bumpBps) / 10000)
		if required <= oldFee {
			required = oldFee + 1
		}
	}

	return newFee >= required
}

func betterPriority(a *types.Transaction, b *types.Transaction) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}

	aFee := a.EffectiveFee()
	bFee := b.EffectiveFee()

	if aFee != bFee {
		return aFee > bFee
	}
	if a.Timestamp != b.Timestamp {
		return a.Timestamp < b.Timestamp
	}
	return a.Hash < b.Hash
}

func worsePriority(a *types.Transaction, b *types.Transaction) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}

	aFee := a.EffectiveFee()
	bFee := b.EffectiveFee()

	if aFee != bFee {
		return aFee < bFee
	}
	if a.Timestamp != b.Timestamp {
		return a.Timestamp > b.Timestamp
	}
	return a.Hash > b.Hash
}

func (p *Pool) MaxPoolSize() int {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.maxPoolSize
}

func (p *Pool) MaxPerSender() int {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.maxPerSender
}

func (p *Pool) MaxFutureNonceGap() types.Nonce {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.maxFutureNonceGap
}

func (p *Pool) ReplacementBumpBps() uint64 {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.replacementBumpBps
}

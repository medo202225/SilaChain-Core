package miner

import (
	"sync"
	"time"
)

const pendingTTL = 2 * time.Second

type pending struct {
	created    time.Time
	parentHash string
	result     *ExecutableData
	lock       sync.Mutex
}

func (p *pending) resolve(parentHash string) *ExecutableData {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.result == nil {
		return nil
	}
	if parentHash != p.parentHash {
		return nil
	}
	if time.Since(p.created) > pendingTTL {
		return nil
	}
	return p.result
}

func (p *pending) update(parentHash string, result *ExecutableData) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.parentHash = parentHash
	p.result = result
	p.created = time.Now()
}

package consensuslegacy

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"time"

	"silachain/internal/chain"
	"silachain/internal/validator"
	"silachain/pkg/types"
)

type ProposerSelection struct {
	blockchain *chain.Blockchain
	clock      *SlotClock
}

func NewProposerSelection(blockchain *chain.Blockchain, clock *SlotClock) *ProposerSelection {
	return &ProposerSelection{
		blockchain: blockchain,
		clock:      clock,
	}
}

func (p *ProposerSelection) ProposerForCurrentSlot(now time.Time) (types.Address, Slot, Epoch, bool) {
	if p == nil || p.blockchain == nil || p.clock == nil {
		return "", 0, 0, false
	}

	slot := p.clock.CurrentSlot(now)
	epoch := p.clock.EpochForSlot(slot)

	proposer, ok := p.ProposerForSlot(slot)
	if !ok {
		return "", slot, epoch, false
	}

	return proposer, slot, epoch, true
}

func (p *ProposerSelection) ProposerForSlot(slot Slot) (types.Address, bool) {
	if p == nil || p.blockchain == nil || p.clock == nil {
		return "", false
	}

	epoch := p.clock.EpochForSlot(slot)
	table := p.epochProposerTable(epoch)
	if len(table) == 0 {
		return "", false
	}

	index := int(uint64(slot) % uint64(len(table)))
	return table[index], true
}

func (p *ProposerSelection) LocalValidatorCanPropose(now time.Time, localValidator types.Address) (types.Address, Slot, Epoch, bool) {
	proposer, slot, epoch, ok := p.ProposerForCurrentSlot(now)
	if !ok {
		return "", slot, epoch, false
	}
	if proposer != localValidator {
		return proposer, slot, epoch, false
	}
	return proposer, slot, epoch, true
}

func (p *ProposerSelection) epochProposerTable(epoch Epoch) []types.Address {
	if p == nil || p.blockchain == nil || p.clock == nil {
		return nil
	}

	weighted := p.blockchain.WeightedValidators()
	if len(weighted) == 0 {
		return nil
	}

	seed := epochSeed(epoch)
	type scored struct {
		Index   int
		Address types.Address
		Score   [32]byte
	}

	scoredSet := make([]scored, 0, len(weighted))
	for i, m := range weighted {
		score := proposerScore(seed, m.Address, i)
		scoredSet = append(scoredSet, scored{
			Index:   i,
			Address: m.Address,
			Score:   score,
		})
	}

	sort.SliceStable(scoredSet, func(i, j int) bool {
		return lessScore(scoredSet[i].Score, scoredSet[j].Score)
	})

	table := make([]types.Address, 0, len(scoredSet))
	for _, item := range scoredSet {
		table = append(table, item.Address)
	}
	return table
}

func PickWeightedProposerForSlot(weighted []validator.Member, slot Slot) (types.Address, bool) {
	if len(weighted) == 0 {
		return "", false
	}

	epoch := Epoch(uint64(slot) / 32)
	seed := epochSeed(epoch)

	type scored struct {
		Index   int
		Address types.Address
		Score   [32]byte
	}

	scoredSet := make([]scored, 0, len(weighted))
	for i, m := range weighted {
		score := proposerScore(seed, m.Address, i)
		scoredSet = append(scoredSet, scored{
			Index:   i,
			Address: m.Address,
			Score:   score,
		})
	}

	sort.SliceStable(scoredSet, func(i, j int) bool {
		return lessScore(scoredSet[i].Score, scoredSet[j].Score)
	})

	index := int(uint64(slot) % uint64(len(scoredSet)))
	return scoredSet[index].Address, true
}

func epochSeed(epoch Epoch) [32]byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(epoch))
	return sha256.Sum256(buf[:])
}

func proposerScore(seed [32]byte, address types.Address, index int) [32]byte {
	payload := make([]byte, 0, len(seed)+len(address)+8)
	payload = append(payload, seed[:]...)
	payload = append(payload, []byte(address)...)

	var idx [8]byte
	binary.BigEndian.PutUint64(idx[:], uint64(index))
	payload = append(payload, idx[:]...)

	return sha256.Sum256(payload)
}

func lessScore(a, b [32]byte) bool {
	for i := 0; i < len(a); i++ {
		if a[i] == b[i] {
			continue
		}
		return a[i] < b[i]
	}
	return false
}

package txpool

import (
	"container/heap"
	"errors"
	"fmt"
	"sort"
)

var (
	ErrNilPool                = errors.New("txpool: nil pool")
	ErrEmptyFrom              = errors.New("txpool: empty from")
	ErrZeroGasLimit           = errors.New("txpool: zero gas limit")
	ErrUnderStateNonce        = errors.New("txpool: tx nonce below sender state nonce")
	ErrReplacementUnderpriced = errors.New("txpool: replacement transaction underpriced")
)

type Tx struct {
	Hash                 string
	From                 string
	Nonce                uint64
	GasLimit             uint64
	MaxFeePerGas         uint64
	MaxPriorityFeePerGas uint64
	Timestamp            int64
}

func (tx Tx) Validate() error {
	if tx.From == "" {
		return ErrEmptyFrom
	}
	if tx.GasLimit == 0 {
		return ErrZeroGasLimit
	}
	return nil
}

func (tx Tx) EffectiveFee(baseFee uint64) uint64 {
	tipCap := tx.MaxPriorityFeePerGas
	feeCap := tx.MaxFeePerGas

	if feeCap <= baseFee {
		return feeCap
	}

	maxPayable := baseFee + tipCap
	if maxPayable < feeCap {
		return maxPayable
	}
	return feeCap
}

type Pool struct {
	baseFee    uint64
	stateNonce map[string]uint64
	pending    map[string]map[uint64]Tx
	count      int
}

func NewPool(baseFee uint64) *Pool {
	return &Pool{
		baseFee:    baseFee,
		stateNonce: make(map[string]uint64),
		pending:    make(map[string]map[uint64]Tx),
	}
}

func (p *Pool) BaseFee() uint64 {
	if p == nil {
		return 0
	}
	return p.baseFee
}

func (p *Pool) SetBaseFee(baseFee uint64) error {
	if p == nil {
		return ErrNilPool
	}
	p.baseFee = baseFee
	return nil
}

func (p *Pool) SetSenderStateNonce(from string, nonce uint64) error {
	if p == nil {
		return ErrNilPool
	}
	if from == "" {
		return ErrEmptyFrom
	}
	p.stateNonce[from] = nonce
	p.pruneSenderBelowStateNonce(from)
	return nil
}

func (p *Pool) SenderStateNonce(from string) uint64 {
	if p == nil {
		return 0
	}
	return p.stateNonce[from]
}

func (p *Pool) PendingCount() int {
	if p == nil {
		return 0
	}
	return p.count
}

func (p *Pool) Add(tx Tx) error {
	if p == nil {
		return ErrNilPool
	}
	if err := tx.Validate(); err != nil {
		return err
	}

	senderStateNonce := p.stateNonce[tx.From]
	if tx.Nonce < senderStateNonce {
		return fmt.Errorf("%w: from=%s tx_nonce=%d state_nonce=%d", ErrUnderStateNonce, tx.From, tx.Nonce, senderStateNonce)
	}

	if _, ok := p.pending[tx.From]; !ok {
		p.pending[tx.From] = make(map[uint64]Tx)
	}

	if existing, ok := p.pending[tx.From][tx.Nonce]; ok {
		if tx.EffectiveFee(p.baseFee) <= existing.EffectiveFee(p.baseFee) {
			return fmt.Errorf(
				"%w: from=%s nonce=%d old_effective_fee=%d new_effective_fee=%d",
				ErrReplacementUnderpriced,
				tx.From,
				tx.Nonce,
				existing.EffectiveFee(p.baseFee),
				tx.EffectiveFee(p.baseFee),
			)
		}
		p.pending[tx.From][tx.Nonce] = tx
		return nil
	}

	p.pending[tx.From][tx.Nonce] = tx
	p.count++
	return nil
}

func (p *Pool) RemoveIncluded(txs []Tx) error {
	if p == nil {
		return ErrNilPool
	}

	for _, tx := range txs {
		nonceMap, ok := p.pending[tx.From]
		if !ok {
			continue
		}

		if _, exists := nonceMap[tx.Nonce]; exists {
			delete(nonceMap, tx.Nonce)
			if p.count > 0 {
				p.count--
			}
		}

		nextNonce := tx.Nonce + 1
		if nextNonce > p.stateNonce[tx.From] {
			p.stateNonce[tx.From] = nextNonce
		}

		p.pruneSenderBelowStateNonce(tx.From)

		if len(nonceMap) == 0 {
			delete(p.pending, tx.From)
		}
	}

	return nil
}

func (p *Pool) Ordered() []Tx {
	if p == nil {
		return nil
	}

	perSender := make(map[string][]Tx, len(p.pending))
	for from, nonceMap := range p.pending {
		list := make([]Tx, 0, len(nonceMap))
		for _, tx := range nonceMap {
			list = append(list, tx)
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Nonce != list[j].Nonce {
				return list[i].Nonce < list[j].Nonce
			}
			leftFee := list[i].EffectiveFee(p.baseFee)
			rightFee := list[j].EffectiveFee(p.baseFee)
			if leftFee != rightFee {
				return leftFee > rightFee
			}
			return list[i].Timestamp < list[j].Timestamp
		})
		perSender[from] = list
	}

	cursors := make(map[string]*senderCursor, len(perSender))
	pq := make(txPriorityQueue, 0, len(perSender))

	for from, list := range perSender {
		cur := &senderCursor{
			from:      from,
			nextNonce: p.stateNonce[from],
			txs:       list,
			index:     0,
		}
		cursors[from] = cur

		if tx, ok := cur.nextExecutable(); ok {
			heap.Push(&pq, &txCandidate{
				tx:      tx,
				from:    from,
				baseFee: p.baseFee,
			})
		}
	}

	out := make([]Tx, 0, p.count)
	for pq.Len() > 0 {
		best := heap.Pop(&pq).(*txCandidate)
		out = append(out, best.tx)

		cur := cursors[best.from]
		cur.nextNonce++

		if nextTx, ok := cur.nextExecutable(); ok {
			heap.Push(&pq, &txCandidate{
				tx:      nextTx,
				from:    best.from,
				baseFee: p.baseFee,
			})
		}
	}

	return out
}

func (p *Pool) pruneSenderBelowStateNonce(from string) {
	nonceMap, ok := p.pending[from]
	if !ok {
		return
	}

	stateNonce := p.stateNonce[from]
	for nonce := range nonceMap {
		if nonce < stateNonce {
			delete(nonceMap, nonce)
			if p.count > 0 {
				p.count--
			}
		}
	}

	if len(nonceMap) == 0 {
		delete(p.pending, from)
	}
}

type senderCursor struct {
	from      string
	nextNonce uint64
	txs       []Tx
	index     int
}

func (c *senderCursor) nextExecutable() (Tx, bool) {
	for c.index < len(c.txs) {
		tx := c.txs[c.index]

		if tx.Nonce < c.nextNonce {
			c.index++
			continue
		}
		if tx.Nonce == c.nextNonce {
			c.index++
			return tx, true
		}

		return Tx{}, false
	}

	return Tx{}, false
}

type txCandidate struct {
	tx      Tx
	from    string
	baseFee uint64
}

type txPriorityQueue []*txCandidate

func (pq txPriorityQueue) Len() int { return len(pq) }

func (pq txPriorityQueue) Less(i, j int) bool {
	leftFee := pq[i].tx.EffectiveFee(pq[i].baseFee)
	rightFee := pq[j].tx.EffectiveFee(pq[j].baseFee)

	if leftFee != rightFee {
		return leftFee > rightFee
	}
	if pq[i].tx.Timestamp != pq[j].tx.Timestamp {
		return pq[i].tx.Timestamp < pq[j].tx.Timestamp
	}
	if pq[i].tx.Nonce != pq[j].tx.Nonce {
		return pq[i].tx.Nonce < pq[j].tx.Nonce
	}
	return pq[i].from < pq[j].from
}

func (pq txPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *txPriorityQueue) Push(x any) {
	*pq = append(*pq, x.(*txCandidate))
}

func (pq *txPriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

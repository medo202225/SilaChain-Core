package miner

import (
	"container/heap"
	"time"

	"silachain/internal/consensus/txpool"
)

type txWithMinerFee struct {
	tx   txpool.Tx
	from string
	fees uint64
}

func newTxWithMinerFee(tx txpool.Tx, baseFee uint64) *txWithMinerFee {
	tip := tx.EffectiveFee(baseFee)
	if tip > baseFee {
		tip = tip - baseFee
	} else {
		tip = 0
	}
	return &txWithMinerFee{
		tx:   tx,
		from: tx.From,
		fees: tip,
	}
}

type txByPriceAndTime []*txWithMinerFee

func (s txByPriceAndTime) Len() int { return len(s) }

func (s txByPriceAndTime) Less(i, j int) bool {
	if s[i].fees == s[j].fees {
		return time.Unix(s[i].tx.Timestamp, 0).Before(time.Unix(s[j].tx.Timestamp, 0))
	}
	return s[i].fees > s[j].fees
}

func (s txByPriceAndTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *txByPriceAndTime) Push(x any) {
	*s = append(*s, x.(*txWithMinerFee))
}

func (s *txByPriceAndTime) Pop() any {
	old := *s
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*s = old[:n-1]
	return x
}

type transactionsByPriceAndNonce struct {
	txs     map[string][]txpool.Tx
	heads   txByPriceAndTime
	baseFee uint64
}

func newTransactionsByPriceAndNonce(txs map[string][]txpool.Tx, baseFee uint64) *transactionsByPriceAndNonce {
	heads := make(txByPriceAndTime, 0, len(txs))
	for from, accTxs := range txs {
		if len(accTxs) == 0 {
			delete(txs, from)
			continue
		}
		heads = append(heads, newTxWithMinerFee(accTxs[0], baseFee))
		txs[from] = accTxs[1:]
	}
	heap.Init(&heads)

	return &transactionsByPriceAndNonce{
		txs:     txs,
		heads:   heads,
		baseFee: baseFee,
	}
}

func (t *transactionsByPriceAndNonce) Peek() (*txpool.Tx, uint64) {
	if len(t.heads) == 0 {
		return nil, 0
	}
	return &t.heads[0].tx, t.heads[0].fees
}

func (t *transactionsByPriceAndNonce) Shift() {
	acc := t.heads[0].from
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = newTxWithMinerFee(txs[0], t.baseFee), txs[1:]
		heap.Fix(&t.heads, 0)
		return
	}
	heap.Pop(&t.heads)
}

func (t *transactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}

func (t *transactionsByPriceAndNonce) Empty() bool {
	return len(t.heads) == 0
}

func (t *transactionsByPriceAndNonce) Clear() {
	t.heads, t.txs = nil, nil
}

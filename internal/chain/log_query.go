package chain

import (
	"strings"

	"silachain/pkg/types"
)

type LogQuery struct {
	Address   string
	Event     string
	Topic0    string
	Topic1    string
	Topic2    string
	Topic3    string
	FromBlock uint64
	ToBlock   uint64
}

type LogRecord struct {
	BlockHeight      uint64            `json:"block_height"`
	BlockHash        types.Hash        `json:"block_hash"`
	TxHash           types.Hash        `json:"tx_hash"`
	TransactionIndex uint64            `json:"transaction_index"`
	LogIndex         uint64            `json:"log_index"`
	Address          string            `json:"address"`
	Event            string            `json:"event"`
	Topics           []string          `json:"topics,omitempty"`
	Data             map[string]string `json:"data,omitempty"`
}

func topicMatches(topics []string, index int, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	if index >= len(topics) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(topics[index]), expected)
}

func (bc *Blockchain) QueryLogs(q LogQuery) []LogRecord {
	if bc == nil || len(bc.blocks) == 0 {
		return nil
	}

	from := q.FromBlock
	to := q.ToBlock

	if to == 0 || to >= uint64(len(bc.blocks)) {
		to = uint64(len(bc.blocks) - 1)
	}
	if from > to {
		return nil
	}

	wantAddress := strings.TrimSpace(q.Address)
	wantEvent := strings.TrimSpace(q.Event)

	out := make([]LogRecord, 0)

	for height := from; height <= to; height++ {
		b := bc.blocks[height]
		if b == nil {
			continue
		}

		for txIndex := range b.Transactions {
			txHash := b.Transactions[txIndex].Hash

			fullReceipt, ok := bc.receipts[string(txHash)]
			if !ok {
				continue
			}

			for logIndex, evt := range fullReceipt.Logs {
				if wantAddress != "" && !strings.EqualFold(strings.TrimSpace(evt.Address), wantAddress) {
					continue
				}
				if wantEvent != "" && !strings.EqualFold(strings.TrimSpace(evt.Name), wantEvent) {
					continue
				}
				if !topicMatches(evt.Topics, 0, q.Topic0) {
					continue
				}
				if !topicMatches(evt.Topics, 1, q.Topic1) {
					continue
				}
				if !topicMatches(evt.Topics, 2, q.Topic2) {
					continue
				}
				if !topicMatches(evt.Topics, 3, q.Topic3) {
					continue
				}

				out = append(out, LogRecord{
					BlockHeight:      uint64(b.Header.Height),
					BlockHash:        b.Header.Hash,
					TxHash:           txHash,
					TransactionIndex: uint64(txIndex),
					LogIndex:         uint64(logIndex),
					Address:          evt.Address,
					Event:            evt.Name,
					Topics:           append([]string(nil), evt.Topics...),
					Data:             evt.Data,
				})
			}
		}
	}

	return out
}

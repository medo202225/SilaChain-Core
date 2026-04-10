package mempool

import (
	"sort"

	"silachain/internal/core/types"
)

func OrderedPending(p *Pool) []types.Transaction {
	pending := p.Pending()

	sort.Slice(pending, func(i, j int) bool {
		if pending[i].From == pending[j].From {
			if pending[i].Nonce != pending[j].Nonce {
				return pending[i].Nonce < pending[j].Nonce
			}

			iFee := pending[i].EffectiveFee()
			jFee := pending[j].EffectiveFee()

			if iFee != jFee {
				return iFee > jFee
			}
			if pending[i].Timestamp != pending[j].Timestamp {
				return pending[i].Timestamp < pending[j].Timestamp
			}
			return pending[i].Hash < pending[j].Hash
		}

		iFee := pending[i].EffectiveFee()
		jFee := pending[j].EffectiveFee()

		if iFee != jFee {
			return iFee > jFee
		}
		if pending[i].Timestamp != pending[j].Timestamp {
			return pending[i].Timestamp < pending[j].Timestamp
		}
		return pending[i].Hash < pending[j].Hash
	})

	return pending
}

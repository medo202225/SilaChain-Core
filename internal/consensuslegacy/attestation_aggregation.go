package consensuslegacy

import "silachain/pkg/types"

type AttestationAggregate struct {
	Slot       Slot
	Epoch      Epoch
	BlockHash  string
	Validators []types.Address
	VoteCount  int
}

func AggregateAttestations(attestations []Attestation) []AttestationAggregate {
	type key struct {
		slot      Slot
		epoch     Epoch
		blockHash string
	}

	grouped := make(map[key]map[types.Address]struct{})

	for _, a := range attestations {
		k := key{
			slot:      a.Slot,
			epoch:     a.Epoch,
			blockHash: a.BlockHash,
		}
		if grouped[k] == nil {
			grouped[k] = make(map[types.Address]struct{})
		}
		grouped[k][a.Validator] = struct{}{}
	}

	out := make([]AttestationAggregate, 0, len(grouped))
	for k, validatorSet := range grouped {
		validators := make([]types.Address, 0, len(validatorSet))
		for v := range validatorSet {
			validators = append(validators, v)
		}

		out = append(out, AttestationAggregate{
			Slot:       k.slot,
			Epoch:      k.epoch,
			BlockHash:  k.blockHash,
			Validators: validators,
			VoteCount:  len(validators),
		})
	}

	return out
}

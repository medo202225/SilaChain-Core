package consensuslegacy

type QuorumResult struct {
	VoteCount            int
	TotalValidators      int
	HasQuorum            bool
	ThresholdNumerator   int
	ThresholdDenominator int
}

func CheckQuorum(voteCount int, totalValidators int) QuorumResult {
	result := QuorumResult{
		VoteCount:            voteCount,
		TotalValidators:      totalValidators,
		ThresholdNumerator:   2,
		ThresholdDenominator: 3,
	}

	if totalValidators <= 0 {
		result.HasQuorum = false
		return result
	}

	result.HasQuorum = voteCount*3 >= totalValidators*2
	return result
}

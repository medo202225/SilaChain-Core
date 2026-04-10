package vm

type ExitReason uint8

const (
	ExitReasonSuccess ExitReason = iota
	ExitReasonRevert
	ExitReasonFault
)

func (r ExitReason) String() string {
	switch r {
	case ExitReasonSuccess:
		return "success"
	case ExitReasonRevert:
		return "revert"
	case ExitReasonFault:
		return "fault"
	default:
		return "unknown"
	}
}

type LogEntry struct {
	Address string
	Topics  []string
	Data    []byte
}

type ExecutionResult struct {
	Reason         ExitReason
	ReturnData     []byte
	RevertData     []byte
	GasUsed        uint64
	GasRemaining   uint64
	Logs           []LogEntry
	CreatedAddress string
	Err            error
}

func (r ExecutionResult) Succeeded() bool {
	return r.Reason == ExitReasonSuccess && r.Err == nil
}

func (r ExecutionResult) Reverted() bool {
	return r.Reason == ExitReasonRevert
}

func (r ExecutionResult) Faulted() bool {
	return r.Reason == ExitReasonFault || r.Err != nil
}

func SuccessResult(returnData []byte, gasUsed, gasRemaining uint64, logs []LogEntry, createdAddress string) ExecutionResult {
	return ExecutionResult{
		Reason:         ExitReasonSuccess,
		ReturnData:     cloneBytes(returnData),
		GasUsed:        gasUsed,
		GasRemaining:   gasRemaining,
		Logs:           cloneLogs(logs),
		CreatedAddress: createdAddress,
	}
}

func RevertResult(revertData []byte, gasUsed, gasRemaining uint64, createdAddress string) ExecutionResult {
	return ExecutionResult{
		Reason:         ExitReasonRevert,
		RevertData:     cloneBytes(revertData),
		GasUsed:        gasUsed,
		GasRemaining:   gasRemaining,
		CreatedAddress: createdAddress,
	}
}

func FaultResult(err error, gasUsed, gasRemaining uint64, createdAddress string) ExecutionResult {
	return ExecutionResult{
		Reason:         ExitReasonFault,
		GasUsed:        gasUsed,
		GasRemaining:   gasRemaining,
		CreatedAddress: createdAddress,
		Err:            err,
	}
}

func cloneBytes(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}

func cloneLogs(in []LogEntry) []LogEntry {
	if len(in) == 0 {
		return nil
	}

	out := make([]LogEntry, len(in))
	for i := range in {
		out[i] = LogEntry{
			Address: in[i].Address,
			Topics:  append([]string(nil), in[i].Topics...),
			Data:    cloneBytes(in[i].Data),
		}
	}
	return out
}

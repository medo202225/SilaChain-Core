package engine

type EngineAPIError struct {
	code int
	msg  string
	err  error
}

func (e *EngineAPIError) ErrorCode() int { return e.code }
func (e *EngineAPIError) Error() string  { return e.msg }

func (e *EngineAPIError) ErrorData() interface{} {
	if e.err == nil {
		return nil
	}
	return struct {
		Error string `json:"err"`
	}{e.err.Error()}
}

func (e *EngineAPIError) With(err error) *EngineAPIError {
	return &EngineAPIError{
		code: e.code,
		msg:  e.msg,
		err:  err,
	}
}

var (
	VALID    = "VALID"
	INVALID  = "INVALID"
	SYNCING  = "SYNCING"
	ACCEPTED = "ACCEPTED"

	GenericServerError       = &EngineAPIError{code: -32000, msg: "Server error"}
	UnknownPayload           = &EngineAPIError{code: -38001, msg: "Unknown payload"}
	InvalidForkChoiceState   = &EngineAPIError{code: -38002, msg: "Invalid forkchoice state"}
	InvalidPayloadAttributes = &EngineAPIError{code: -38003, msg: "Invalid payload attributes"}
	TooLargeRequest          = &EngineAPIError{code: -38004, msg: "Too large request"}
	InvalidParams            = &EngineAPIError{code: -32602, msg: "Invalid parameters"}
	UnsupportedFork          = &EngineAPIError{code: -38005, msg: "Unsupported fork"}

	STATUS_INVALID = ForkChoiceResponse{
		PayloadStatus: PayloadStatusV1{
			Status: INVALID,
		},
		PayloadID: nil,
	}

	STATUS_SYNCING = ForkChoiceResponse{
		PayloadStatus: PayloadStatusV1{
			Status: SYNCING,
		},
		PayloadID: nil,
	}

	INVALID_TERMINAL_BLOCK = PayloadStatusV1{
		Status:          INVALID,
		LatestValidHash: stringPtr(""),
	}
)

func stringPtr(v string) *string {
	return &v
}

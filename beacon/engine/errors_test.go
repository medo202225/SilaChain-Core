package engine

import "testing"

func TestEngineAPIError_WithAndData(t *testing.T) {
	base := InvalidPayloadAttributes
	wrapped := base.With(assertErr("bad attrs"))

	if wrapped.ErrorCode() != -38003 {
		t.Fatalf("unexpected code: got=%d want=-38003", wrapped.ErrorCode())
	}
	if wrapped.Error() != "Invalid payload attributes" {
		t.Fatalf("unexpected message: got=%s", wrapped.Error())
	}

	data, ok := wrapped.ErrorData().(struct {
		Error string `json:"err"`
	})
	if !ok {
		t.Fatalf("expected structured error data")
	}
	if data.Error != "bad attrs" {
		t.Fatalf("unexpected error data: got=%s want=bad attrs", data.Error)
	}
}

func TestStatusConstants(t *testing.T) {
	if VALID != "VALID" || INVALID != "INVALID" || SYNCING != "SYNCING" || ACCEPTED != "ACCEPTED" {
		t.Fatalf("unexpected status constants")
	}
}

func TestPredefinedErrors(t *testing.T) {
	if UnknownPayload.ErrorCode() != -38001 {
		t.Fatalf("unexpected unknown payload code: %d", UnknownPayload.ErrorCode())
	}
	if InvalidForkChoiceState.ErrorCode() != -38002 {
		t.Fatalf("unexpected invalid forkchoice state code: %d", InvalidForkChoiceState.ErrorCode())
	}
	if TooLargeRequest.ErrorCode() != -38004 {
		t.Fatalf("unexpected too large request code: %d", TooLargeRequest.ErrorCode())
	}
	if UnsupportedFork.ErrorCode() != -38005 {
		t.Fatalf("unexpected unsupported fork code: %d", UnsupportedFork.ErrorCode())
	}
}

func TestPredefinedStatuses(t *testing.T) {
	if STATUS_INVALID.PayloadStatus.Status != INVALID {
		t.Fatalf("unexpected invalid status payload: %s", STATUS_INVALID.PayloadStatus.Status)
	}
	if STATUS_SYNCING.PayloadStatus.Status != SYNCING {
		t.Fatalf("unexpected syncing status payload: %s", STATUS_SYNCING.PayloadStatus.Status)
	}
	if INVALID_TERMINAL_BLOCK.Status != INVALID {
		t.Fatalf("unexpected invalid terminal block status: %s", INVALID_TERMINAL_BLOCK.Status)
	}
	if INVALID_TERMINAL_BLOCK.LatestValidHash == nil {
		t.Fatalf("expected latest valid hash pointer")
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

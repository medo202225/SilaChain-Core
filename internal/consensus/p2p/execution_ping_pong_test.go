package p2p

import (
	"testing"
	"time"
)

func TestExecutionSession_RejectsPingBeforeReady(t *testing.T) {
	session := NewExecutionSession(nil)

	ping, err := EncodeExecutionMessage(
		ExecutionMessageCodePing,
		ExecutionPingMessage{Timestamp: time.Now().UTC().Unix()},
	)
	if err != nil {
		t.Fatalf("encode ping: %v", err)
	}

	err = session.HandleIncomingMessage(ping)
	if err == nil {
		t.Fatal("expected error for ping before ready")
	}
	if err != ErrExecutionReadyRequiredBeforePing {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecutionSession_AcceptsPingAfterReadyAndSendsPong(t *testing.T) {
	var sent [][]byte

	session := NewExecutionSession(func(data []byte) error {
		copyData := make([]byte, len(data))
		copy(copyData, data)
		sent = append(sent, copyData)
		return nil
	})

	status, err := EncodeExecutionMessage(
		ExecutionMessageCodeStatus,
		ExecutionStatusMessage{
			NetworkID:    "sila-mainnet",
			Version:      "1",
			Capabilities: []string{"execution"},
		},
	)
	if err != nil {
		t.Fatalf("encode status: %v", err)
	}

	hello, err := EncodeExecutionMessage(
		ExecutionMessageCodeHello,
		ExecutionHelloMessage{
			ProtocolName:    "SilaChain/native-exec-protocol",
			ProtocolVersion: "1",
			NetworkID:       "sila-mainnet",
			Capabilities:    []string{"execution"},
			ClientName:      "silachain",
		},
	)
	if err != nil {
		t.Fatalf("encode hello: %v", err)
	}

	ready, err := EncodeExecutionMessage(
		ExecutionMessageCodeReady,
		ExecutionReadyMessage{
			Ready: true,
		},
	)
	if err != nil {
		t.Fatalf("encode ready: %v", err)
	}

	ts := time.Now().UTC().Unix()

	ping, err := EncodeExecutionMessage(
		ExecutionMessageCodePing,
		ExecutionPingMessage{
			Timestamp: ts,
		},
	)
	if err != nil {
		t.Fatalf("encode ping: %v", err)
	}

	if err := session.HandleIncomingMessage(status); err != nil {
		t.Fatalf("handle status: %v", err)
	}
	if err := session.HandleIncomingMessage(hello); err != nil {
		t.Fatalf("handle hello: %v", err)
	}
	if err := session.HandleIncomingMessage(ready); err != nil {
		t.Fatalf("handle ready: %v", err)
	}
	if !session.IsReady() {
		t.Fatal("expected session to be ready")
	}

	if err := session.HandleIncomingMessage(ping); err != nil {
		t.Fatalf("handle ping: %v", err)
	}

	if session.LastPingTime().Unix() != ts {
		t.Fatalf("unexpected last ping timestamp: %d", session.LastPingTime().Unix())
	}
	if session.LastPongTime().Unix() != ts {
		t.Fatalf("unexpected last pong timestamp: %d", session.LastPongTime().Unix())
	}

	if len(sent) != 1 {
		t.Fatalf("expected 1 outgoing pong, got %d", len(sent))
	}

	envelope, err := DecodeExecutionEnvelope(sent[0])
	if err != nil {
		t.Fatalf("decode outgoing envelope: %v", err)
	}
	if envelope.Code != ExecutionMessageCodePong {
		t.Fatalf("unexpected outgoing code: %d", envelope.Code)
	}

	pong, err := DecodeExecutionPongMessage(envelope.Payload)
	if err != nil {
		t.Fatalf("decode pong: %v", err)
	}
	if pong.Timestamp != ts {
		t.Fatalf("unexpected pong timestamp: %d", pong.Timestamp)
	}
}

func TestExecutionSession_SendPingRequiresReady(t *testing.T) {
	session := NewExecutionSession(nil)

	err := session.SendPing(time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for sending ping before ready")
	}
	if err != ErrExecutionReadyRequiredBeforePing {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecutionSession_SendPingAfterReady(t *testing.T) {
	var sent [][]byte

	session := NewExecutionSession(func(data []byte) error {
		copyData := make([]byte, len(data))
		copy(copyData, data)
		sent = append(sent, copyData)
		return nil
	})

	status, _ := EncodeExecutionMessage(
		ExecutionMessageCodeStatus,
		ExecutionStatusMessage{
			NetworkID:    "sila-mainnet",
			Version:      "1",
			Capabilities: []string{"execution"},
		},
	)

	hello, _ := EncodeExecutionMessage(
		ExecutionMessageCodeHello,
		ExecutionHelloMessage{
			ProtocolName:    "SilaChain/native-exec-protocol",
			ProtocolVersion: "1",
			NetworkID:       "sila-mainnet",
			Capabilities:    []string{"execution"},
			ClientName:      "silachain",
		},
	)

	ready, _ := EncodeExecutionMessage(
		ExecutionMessageCodeReady,
		ExecutionReadyMessage{
			Ready: true,
		},
	)

	if err := session.HandleIncomingMessage(status); err != nil {
		t.Fatalf("handle status: %v", err)
	}
	if err := session.HandleIncomingMessage(hello); err != nil {
		t.Fatalf("handle hello: %v", err)
	}
	if err := session.HandleIncomingMessage(ready); err != nil {
		t.Fatalf("handle ready: %v", err)
	}

	ts := time.Now().UTC().Truncate(time.Second)

	if err := session.SendPing(ts); err != nil {
		t.Fatalf("send ping: %v", err)
	}

	if len(sent) != 1 {
		t.Fatalf("expected 1 outgoing ping, got %d", len(sent))
	}

	envelope, err := DecodeExecutionEnvelope(sent[0])
	if err != nil {
		t.Fatalf("decode outgoing envelope: %v", err)
	}
	if envelope.Code != ExecutionMessageCodePing {
		t.Fatalf("unexpected outgoing code: %d", envelope.Code)
	}

	ping, err := DecodeExecutionPingMessage(envelope.Payload)
	if err != nil {
		t.Fatalf("decode ping: %v", err)
	}
	if ping.Timestamp != ts.Unix() {
		t.Fatalf("unexpected ping timestamp: %d", ping.Timestamp)
	}
}

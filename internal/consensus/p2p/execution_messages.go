package p2p

import (
	"encoding/json"
	"errors"
)

const (
	ExecutionMessageCodeStatus uint8 = iota
	ExecutionMessageCodeHello
	ExecutionMessageCodeReady
	ExecutionMessageCodePing
	ExecutionMessageCodePong
)

const (
	ExecutionMsgStatus = ExecutionMessageCodeStatus
	ExecutionMsgHello  = ExecutionMessageCodeHello
	ExecutionMsgReady  = ExecutionMessageCodeReady
	ExecutionMsgPing   = ExecutionMessageCodePing
	ExecutionMsgPong   = ExecutionMessageCodePong
)

var (
	ErrInvalidExecutionMessagePayload = errors.New("invalid execution message payload")
)

type ExecutionEnvelope struct {
	Code    uint8           `json:"code"`
	Payload json.RawMessage `json:"payload"`
}

type ExecutionStatusMessage struct {
	NetworkID    string   `json:"network_id"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

type ExecutionHello struct {
	ProtocolName    string   `json:"protocol_name"`
	ProtocolVersion string   `json:"protocol_version"`
	NetworkID       string   `json:"network_id"`
	Capabilities    []string `json:"capabilities"`
	ClientName      string   `json:"client_name"`
}

type ExecutionHelloMessage = ExecutionHello

type ExecutionReady struct {
	Ready bool `json:"ready"`
}

type ExecutionReadyMessage = ExecutionReady

type ExecutionPing struct {
	Timestamp int64 `json:"timestamp"`
}

type ExecutionPingMessage = ExecutionPing

type ExecutionPong struct {
	Timestamp int64 `json:"timestamp"`
}

type ExecutionPongMessage = ExecutionPong

func EncodeExecutionMessage(code uint8, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	envelope := ExecutionEnvelope{
		Code:    code,
		Payload: raw,
	}

	return json.Marshal(envelope)
}

func DecodeExecutionEnvelope(data []byte) (ExecutionEnvelope, error) {
	var envelope ExecutionEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ExecutionEnvelope{}, err
	}
	return envelope, nil
}

func DecodeExecutionStatusMessage(data []byte) (ExecutionStatusMessage, error) {
	var msg ExecutionStatusMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return ExecutionStatusMessage{}, ErrInvalidExecutionMessagePayload
	}
	return msg, nil
}

func DecodeExecutionHelloMessage(data []byte) (ExecutionHelloMessage, error) {
	var msg ExecutionHelloMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return ExecutionHelloMessage{}, ErrInvalidExecutionMessagePayload
	}
	return msg, nil
}

func DecodeExecutionReadyMessage(data []byte) (ExecutionReadyMessage, error) {
	var msg ExecutionReadyMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return ExecutionReadyMessage{}, ErrInvalidExecutionMessagePayload
	}
	return msg, nil
}

func DecodeExecutionPingMessage(data []byte) (ExecutionPingMessage, error) {
	var msg ExecutionPingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return ExecutionPingMessage{}, ErrInvalidExecutionMessagePayload
	}
	return msg, nil
}

func DecodeExecutionPongMessage(data []byte) (ExecutionPongMessage, error) {
	var msg ExecutionPongMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return ExecutionPongMessage{}, ErrInvalidExecutionMessagePayload
	}
	return msg, nil
}

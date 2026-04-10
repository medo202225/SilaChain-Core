package p2p

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrExecutionStatusRequiredBeforeHello = errors.New("execution status required before hello")
	ErrExecutionHelloRequiredBeforeReady  = errors.New("execution hello required before ready")
	ErrExecutionReadyRequiredBeforePing   = errors.New("execution ready required before ping")
)

type ExecutionSession struct {
	mu sync.RWMutex

	peerID string

	statusSent     bool
	statusReceived bool
	helloSent      bool
	helloReceived  bool
	readySent      bool
	readyReceived  bool

	lastPingTime time.Time
	lastPongTime time.Time

	send func([]byte) error
}

func NewExecutionSession(arg any) *ExecutionSession {
	s := &ExecutionSession{}

	switch v := arg.(type) {
	case string:
		s.peerID = v
	case func([]byte) error:
		s.send = v
	case nil:
	default:
	}

	return s
}

func (s *ExecutionSession) SetSender(send func([]byte) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.send = send
}

func (s *ExecutionSession) PeerID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peerID
}

func (s *ExecutionSession) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readyReceived
}

func (s *ExecutionSession) LastPingTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastPingTime
}

func (s *ExecutionSession) LastPongTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastPongTime
}

func (s *ExecutionSession) MarkStatusSent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusSent = true
}

func (s *ExecutionSession) MarkStatusReceived() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusReceived = true
}

func (s *ExecutionSession) HasStatusExchanged() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusSent && s.statusReceived
}

func (s *ExecutionSession) MarkHelloSent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.helloSent = true
}

func (s *ExecutionSession) MarkHelloReceived() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.helloReceived = true
}

func (s *ExecutionSession) HasHelloExchanged() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.helloSent && s.helloReceived
}

func (s *ExecutionSession) MarkReadySent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readySent = true
}

func (s *ExecutionSession) MarkReadyReceived() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readyReceived = true
}

func (s *ExecutionSession) MarkPing(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastPingTime = t.UTC()
}

func (s *ExecutionSession) MarkPong(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastPongTime = t.UTC()
}

func (s *ExecutionSession) HandleIncomingMessage(data []byte) error {
	envelope, err := DecodeExecutionEnvelope(data)
	if err != nil {
		return err
	}

	switch envelope.Code {
	case ExecutionMessageCodeStatus:
		_, err := DecodeExecutionStatusMessage(envelope.Payload)
		if err != nil {
			return err
		}

		s.mu.Lock()
		s.statusReceived = true
		s.mu.Unlock()
		return nil

	case ExecutionMessageCodeHello:
		_, err := DecodeExecutionHelloMessage(envelope.Payload)
		if err != nil {
			return err
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		if !s.statusReceived {
			return ErrExecutionStatusRequiredBeforeHello
		}

		s.helloReceived = true
		return nil

	case ExecutionMessageCodeReady:
		msg, err := DecodeExecutionReadyMessage(envelope.Payload)
		if err != nil {
			return err
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		if !s.helloReceived {
			return ErrExecutionHelloRequiredBeforeReady
		}

		s.readyReceived = msg.Ready
		return nil

	case ExecutionMessageCodePing:
		msg, err := DecodeExecutionPingMessage(envelope.Payload)
		if err != nil {
			return err
		}

		pingTime := time.Unix(msg.Timestamp, 0).UTC()

		s.mu.Lock()
		ready := s.readyReceived
		if ready {
			s.lastPingTime = pingTime
		}
		s.mu.Unlock()

		if !ready {
			return ErrExecutionReadyRequiredBeforePing
		}

		return s.sendPong(pingTime)

	case ExecutionMessageCodePong:
		msg, err := DecodeExecutionPongMessage(envelope.Payload)
		if err != nil {
			return err
		}

		pongTime := time.Unix(msg.Timestamp, 0).UTC()

		s.mu.Lock()
		s.lastPongTime = pongTime
		s.mu.Unlock()
		return nil

	default:
		return nil
	}
}

func (s *ExecutionSession) SendPing(t time.Time) error {
	s.mu.RLock()
	ready := s.readyReceived
	s.mu.RUnlock()

	if !ready {
		return ErrExecutionReadyRequiredBeforePing
	}

	if s.send == nil {
		return nil
	}

	payload, err := EncodeExecutionMessage(
		ExecutionMessageCodePing,
		ExecutionPingMessage{Timestamp: t.UTC().Unix()},
	)
	if err != nil {
		return err
	}

	return s.send(payload)
}

func (s *ExecutionSession) sendPong(t time.Time) error {
	if s.send == nil {
		return nil
	}

	payload, err := EncodeExecutionMessage(
		ExecutionMessageCodePong,
		ExecutionPongMessage{Timestamp: t.UTC().Unix()},
	)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.lastPongTime = t.UTC()
	s.mu.Unlock()

	return s.send(payload)
}

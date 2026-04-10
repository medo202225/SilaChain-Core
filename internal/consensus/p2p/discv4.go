package p2p

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	chaincrypto "silachain/pkg/crypto"
)

type SilaDiscoveryPacket struct {
	Type    string `json:"type"`
	FromENR string `json:"from_enr"`
	Time    int64  `json:"time"`
}

type SilaDiscoveryPeer struct {
	PeerID   string
	ENRText  string
	IP       string
	UDP      int
	TCP      int
	LastSeen time.Time
}

type SilaDiscoveryService struct {
	cfg       *Config
	identity  *Identity
	canonical *CanonicalENR

	conn     *net.UDPConn
	selfText string

	mu    sync.RWMutex
	peers map[string]SilaDiscoveryPeer
}

func StartSilaDiscovery(cfg *Config, identity *Identity, canonical *CanonicalENR) (*SilaDiscoveryService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil p2p config")
	}
	if identity == nil {
		return nil, fmt.Errorf("nil identity")
	}
	if canonical == nil || canonical.Sila == nil {
		return nil, fmt.Errorf("nil sila enr")
	}

	selfText, err := canonical.Sila.Text()
	if err != nil {
		return nil, fmt.Errorf("build self sila enr text: %w", err)
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.ListenIP, cfg.UDPPort))
	if err != nil {
		return nil, fmt.Errorf("resolve udp addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}

	svc := &SilaDiscoveryService{
		cfg:       cfg,
		identity:  identity,
		canonical: canonical,
		conn:      conn,
		selfText:  selfText,
		peers:     make(map[string]SilaDiscoveryPeer),
	}

	go svc.readLoop()

	for _, bootnode := range cfg.Bootnodes {
		bootnode := bootnode
		go func() {
			_ = svc.PingENRText(bootnode)
		}()
	}

	return svc, nil
}

func (s *SilaDiscoveryService) Close() {
	if s == nil || s.conn == nil {
		return
	}
	_ = s.conn.Close()
}

func (s *SilaDiscoveryService) SelfText() string {
	if s == nil {
		return ""
	}
	return s.selfText
}

func (s *SilaDiscoveryService) SelfRecord() *SilaENR {
	if s == nil || s.canonical == nil {
		return nil
	}
	return s.canonical.Sila
}

func (s *SilaDiscoveryService) TableNodeCount() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.peers)
}

func (s *SilaDiscoveryService) ResolvePeer(peerID string) *SilaDiscoveryPeer {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	peer, ok := s.peers[peerID]
	if !ok {
		return nil
	}

	copyPeer := peer
	return &copyPeer
}

func (s *SilaDiscoveryService) PingENRText(enrText string) error {
	record, err := ParseSilaENRText(enrText)
	if err != nil {
		return fmt.Errorf("parse bootnode enr: %w", err)
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", record.IP, record.UDP))
	if err != nil {
		return fmt.Errorf("resolve bootnode udp addr: %w", err)
	}

	packet := SilaDiscoveryPacket{
		Type:    "ping",
		FromENR: s.selfText,
		Time:    time.Now().UTC().Unix(),
	}

	return s.writePacket(packet, addr)
}

func (s *SilaDiscoveryService) readLoop() {
	buf := make([]byte, 8192)

	for {
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		var packet SilaDiscoveryPacket
		if err := json.Unmarshal(buf[:n], &packet); err != nil {
			continue
		}

		record, err := ParseSilaENRText(packet.FromENR)
		if err != nil {
			continue
		}

		s.upsertPeer(record, packet.FromENR, addr)

		switch packet.Type {
		case "ping":
			resp := SilaDiscoveryPacket{
				Type:    "pong",
				FromENR: s.selfText,
				Time:    time.Now().UTC().Unix(),
			}
			_ = s.writePacket(resp, addr)
		case "pong":
			// peer already stored above
		}
	}
}

func (s *SilaDiscoveryService) upsertPeer(record *SilaENR, enrText string, addr *net.UDPAddr) {
	if record == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.peers[record.PeerID] = SilaDiscoveryPeer{
		PeerID:   record.PeerID,
		ENRText:  enrText,
		IP:       record.IP,
		UDP:      record.UDP,
		TCP:      record.TCP,
		LastSeen: time.Now().UTC(),
	}
	if addr != nil && record.IP == "" {
		peer := s.peers[record.PeerID]
		peer.IP = addr.IP.String()
		s.peers[record.PeerID] = peer
	}
}

func (s *SilaDiscoveryService) writePacket(packet SilaDiscoveryPacket, addr *net.UDPAddr) error {
	raw, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("marshal discovery packet: %w", err)
	}

	_, err = s.conn.WriteToUDP(raw, addr)
	if err != nil {
		return fmt.Errorf("write discovery packet: %w", err)
	}
	return nil
}

func ParseSilaENRText(text string) (*SilaENR, error) {
	if !strings.HasPrefix(text, "enr:") {
		return nil, fmt.Errorf("invalid sila enr prefix")
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(text, "enr:"))
	if err != nil {
		return nil, fmt.Errorf("decode sila enr text: %w", err)
	}

	var record SilaENR
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("decode sila enr json: %w", err)
	}

	if err := VerifySilaENR(&record); err != nil {
		return nil, err
	}

	return &record, nil
}

func VerifySilaENR(record *SilaENR) error {
	if record == nil {
		return fmt.Errorf("nil sila enr")
	}
	if record.Signature == "" {
		return fmt.Errorf("missing sila enr signature")
	}
	if record.PublicKey == "" {
		return fmt.Errorf("missing sila enr public key")
	}

	pub, err := chaincrypto.HexToPublicKey(record.PublicKey)
	if err != nil {
		return fmt.Errorf("parse sila enr public key: %w", err)
	}

	raw, err := canonicalSilaENRJSON(record, true)
	if err != nil {
		return err
	}

	hashHex := chaincrypto.HashBytes(raw)
	ok, err := chaincrypto.VerifyHashHex(pub, hashHex, record.Signature)
	if err != nil {
		return fmt.Errorf("verify sila enr signature: %w", err)
	}
	if !ok {
		return fmt.Errorf("invalid sila enr signature")
	}

	return nil
}

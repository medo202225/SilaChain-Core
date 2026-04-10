package p2p

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sort"

	chaincrypto "silachain/pkg/crypto"
)

type SilaENR struct {
	Sequence    uint64            `json:"seq"`
	IP          string            `json:"ip"`
	TCP         int               `json:"tcp"`
	UDP         int               `json:"udp"`
	PublicKey   string            `json:"secp256k1_public_key"`
	PeerID      string            `json:"peer_id"`
	NetworkName string            `json:"network_name"`
	Fields      map[string]string `json:"fields,omitempty"`
	Signature   string            `json:"signature"`
}

func BuildSilaENR(cfg *Config, identity *Identity) (*SilaENR, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil p2p config")
	}
	if identity == nil || identity.ECDSAPrivateKey == nil {
		return nil, fmt.Errorf("nil identity")
	}

	ip := net.ParseIP(cfg.ListenIP)
	if ip == nil {
		if cfg.ListenIP == "0.0.0.0" {
			ip = net.ParseIP("127.0.0.1")
		} else {
			return nil, fmt.Errorf("invalid listen ip: %s", cfg.ListenIP)
		}
	}

	record := &SilaENR{
		Sequence:    1,
		IP:          ip.String(),
		TCP:         cfg.TCPPort,
		UDP:         cfg.UDPPort,
		PublicKey:   chaincrypto.PublicKeyToHex(&identity.ECDSAPrivateKey.PublicKey),
		PeerID:      identity.PeerID,
		NetworkName: cfg.NetworkName,
		Fields: map[string]string{
			"id":   "v4",
			"sila": cfg.NetworkName,
		},
	}

	sig, err := signSilaENR(identity.ECDSAPrivateKey, record)
	if err != nil {
		return nil, err
	}
	record.Signature = sig

	return record, nil
}

func (e *SilaENR) Text() (string, error) {
	if e == nil {
		return "", fmt.Errorf("nil sila enr")
	}

	raw, err := canonicalSilaENRJSON(e, false)
	if err != nil {
		return "", err
	}

	return "enr:" + base64.RawURLEncoding.EncodeToString(raw), nil
}

func signSilaENR(priv *ecdsa.PrivateKey, record *SilaENR) (string, error) {
	raw, err := canonicalSilaENRJSON(record, true)
	if err != nil {
		return "", err
	}

	hashHex := chaincrypto.HashBytes(raw)
	sigHex, err := chaincrypto.SignHashHex(priv, hashHex)
	if err != nil {
		return "", fmt.Errorf("sign sila enr payload: %w", err)
	}
	return sigHex, nil
}

func canonicalSilaENRJSON(record *SilaENR, zeroSignature bool) ([]byte, error) {
	if record == nil {
		return nil, fmt.Errorf("nil sila enr")
	}

	fields := orderedSilaFields(record.Fields)

	ordered := map[string]any{
		"fields":               fields,
		"ip":                   record.IP,
		"network_name":         record.NetworkName,
		"peer_id":              record.PeerID,
		"secp256k1_public_key": record.PublicKey,
		"seq":                  record.Sequence,
		"tcp":                  record.TCP,
		"udp":                  record.UDP,
	}

	if !zeroSignature && record.Signature != "" {
		ordered["signature"] = record.Signature
	}

	raw, err := json.Marshal(ordered)
	if err != nil {
		return nil, fmt.Errorf("marshal sila enr json: %w", err)
	}

	return raw, nil
}

func orderedSilaFields(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}

	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]string, len(input))
	for _, k := range keys {
		out[k] = input[k]
	}
	return out
}

package p2p

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type noopENRDB struct{}

func (n *noopENRDB) Close() error { return nil }

type CanonicalENR struct {
	DBPath string
	DB     *noopENRDB
	Text   string
	Sila   *SilaENR
}

func BuildCanonicalENR(cfg *Config, identity *Identity) (*CanonicalENR, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil p2p config")
	}
	if identity == nil || identity.ECDSAPrivateKey == nil {
		return nil, fmt.Errorf("nil identity")
	}

	silaRecord, err := BuildSilaENR(cfg, identity)
	if err != nil {
		return nil, err
	}

	text, err := silaRecord.Text()
	if err != nil {
		return nil, err
	}

	dbPath := canonicalDBPath(cfg)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create enr db directory: %w", err)
	}

	return &CanonicalENR{
		DBPath: dbPath,
		DB:     &noopENRDB{},
		Text:   text,
		Sila:   silaRecord,
	}, nil
}

func canonicalDBPath(cfg *Config) string {
	baseDir := filepath.Dir(cfg.KeyFile)
	if baseDir == "." || baseDir == "" {
		baseDir = filepath.Join("runtime", "p2p", sanitizeName(cfg.NetworkName))
	}

	keyBase := strings.TrimSuffix(filepath.Base(cfg.KeyFile), filepath.Ext(cfg.KeyFile))
	keyBase = sanitizeName(keyBase)
	if keyBase == "" {
		keyBase = "node"
	}

	return filepath.Join(baseDir, keyBase+"_enr_db")
}

func sanitizeName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "default"
	}

	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(v)
}

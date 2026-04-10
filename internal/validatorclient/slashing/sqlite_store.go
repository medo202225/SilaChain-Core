package slashing

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db   *sql.DB
	path string
}

func NewSQLiteStore(path string) *SQLiteStore {
	return &SQLiteStore{path: path}
}

func (s *SQLiteStore) Init(ctx context.Context) error {
	if s.path == "" {
		return ErrInvalidInput
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create slashing dir: %w", err)
	}

	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	s.db = db

	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=FULL;`,

		`CREATE TABLE IF NOT EXISTS signed_blocks (
pubkey BLOB NOT NULL,
slot INTEGER NOT NULL,
signing_root BLOB,
created_at TIMESTAMP NOT NULL,
PRIMARY KEY (pubkey, slot)
);`,

		`CREATE TABLE IF NOT EXISTS signed_attestations (
id INTEGER PRIMARY KEY AUTOINCREMENT,
pubkey BLOB NOT NULL,
source_epoch INTEGER NOT NULL,
target_epoch INTEGER NOT NULL,
signing_root BLOB,
created_at TIMESTAMP NOT NULL
);`,

		`CREATE INDEX IF NOT EXISTS idx_att_pubkey_target
 ON signed_attestations(pubkey, target_epoch);`,

		`CREATE INDEX IF NOT EXISTS idx_att_pubkey_source_target
 ON signed_attestations(pubkey, source_epoch, target_epoch);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			_ = s.db.Close()
			return fmt.Errorf("exec schema: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) CheckAndRecordBlock(ctx context.Context, pubKey []byte, slot uint64, signingRoot []byte) error {
	if len(pubKey) == 0 || len(signingRoot) == 0 {
		return ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existingRoot []byte
	err = tx.QueryRowContext(
		ctx,
		`SELECT signing_root FROM signed_blocks WHERE pubkey = ? AND slot = ?`,
		pubKey, slot,
	).Scan(&existingRoot)

	switch {
	case err == sql.ErrNoRows:
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO signed_blocks(pubkey, slot, signing_root, created_at) VALUES (?, ?, ?, ?)`,
			pubKey, slot, signingRoot, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert block record: %w", err)
		}
	case err != nil:
		return fmt.Errorf("query block record: %w", err)
	default:
		if sameBytes(existingRoot, signingRoot) {
			return tx.Commit()
		}
		return ErrSlashableBlock
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit block tx: %w", err)
	}

	return nil
}

func (s *SQLiteStore) CheckAndRecordAttestation(ctx context.Context, pubKey []byte, sourceEpoch uint64, targetEpoch uint64, signingRoot []byte) error {
	if len(pubKey) == 0 || len(signingRoot) == 0 {
		return ErrInvalidInput
	}
	if sourceEpoch > targetEpoch {
		return ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT source_epoch, target_epoch, signing_root
 FROM signed_attestations
 WHERE pubkey = ?`,
		pubKey,
	)
	if err != nil {
		return fmt.Errorf("query attestations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var oldSource uint64
		var oldTarget uint64
		var oldRoot []byte

		if err := rows.Scan(&oldSource, &oldTarget, &oldRoot); err != nil {
			return fmt.Errorf("scan attestation row: %w", err)
		}

		if oldTarget == targetEpoch && !sameBytes(oldRoot, signingRoot) {
			return ErrSlashableAttestation
		}

		if isSurrounding(sourceEpoch, targetEpoch, oldSource, oldTarget) {
			return ErrSlashableAttestation
		}

		if isSurrounded(sourceEpoch, targetEpoch, oldSource, oldTarget) {
			return ErrSlashableAttestation
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate attestations: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO signed_attestations(pubkey, source_epoch, target_epoch, signing_root, created_at)
 VALUES (?, ?, ?, ?, ?)`,
		pubKey, sourceEpoch, targetEpoch, signingRoot, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert attestation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit attestation tx: %w", err)
	}

	return nil
}

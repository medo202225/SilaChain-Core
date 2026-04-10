package storage

import (
	"os"
	"testing"
)

func TestCleanupTempFiles_RemovesTmpFiles(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)

	tmpPath := db.Path("accounts.json.tmp")
	if err := os.WriteFile(tmpPath, []byte(`{"broken":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile tmp failed: %v", err)
	}

	if err := db.CleanupTempFiles(); err != nil {
		t.Fatalf("CleanupTempFiles failed: %v", err)
	}

	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("expected tmp file removed, stat err=%v", err)
	}
}

func TestReadJSON_IgnoresMissingFile(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)

	var out map[string]any
	if err := db.ReadJSON("missing.json", &out); err != nil {
		t.Fatalf("expected nil on missing file, got %v", err)
	}
}

func TestReadJSON_RejectsCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)

	path := db.Path("bad.json")
	if err := os.WriteFile(path, []byte(`{"bad":`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var out map[string]any
	err := db.ReadJSON("bad.json", &out)
	if err != ErrCorruptStorageFile {
		t.Fatalf("expected ErrCorruptStorageFile, got %v", err)
	}
}

func TestReadJSON_CleansStaleTmpBeforeRead(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)

	tmpPath := db.Path("blocks.json.tmp")
	finalPath := db.Path("blocks.json")

	if err := os.WriteFile(tmpPath, []byte(`{"stale":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile tmp failed: %v", err)
	}
	if err := os.WriteFile(finalPath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile final failed: %v", err)
	}

	var out map[string]any
	if err := db.ReadJSON("blocks.json", &out); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}

	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale tmp removed, stat err=%v", err)
	}
	if out["ok"] != true {
		t.Fatalf("expected final file data, got %+v", out)
	}
}

func TestWriteJSONAtomic_WritesReadableJSON(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)

	payload := map[string]any{
		"height": 7,
		"ok":     true,
	}

	if err := db.WriteJSONAtomic("meta.json", payload); err != nil {
		t.Fatalf("WriteJSONAtomic failed: %v", err)
	}

	var out map[string]any
	if err := db.ReadJSON("meta.json", &out); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}

	if out["ok"] != true {
		t.Fatalf("expected ok=true, got %+v", out)
	}
	if out["height"].(float64) != 7 {
		t.Fatalf("expected height=7, got %+v", out)
	}
}

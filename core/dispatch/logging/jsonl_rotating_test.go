package logging

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestRotatingJSONLStore_Rotation(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/log.jsonl"
	store, err := NewRotatingJSONLStore(path, 1, 2, 1)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()
	rec := LogRecord{Timestamp: time.Now()}
	for i := 0; i < 100; i++ {
		if err := store.Append(context.Background(), rec); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	files, _ := filepath.Glob(path + "*")
	if len(files) == 0 {
		t.Fatalf("expected rotated files")
	}
}

func TestRotatingJSONLStore_Query(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/log.jsonl"
	store, err := NewRotatingJSONLStore(path, 1, 2, 1)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()
	now := time.Now()
	rec := LogRecord{Timestamp: now}
	_ = store.Append(context.Background(), rec)
	out, err := store.Query(context.Background(), LogQuery{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected records")
	}
}

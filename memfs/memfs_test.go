package memfs

import (
	"errors"
	"testing"
)

func TestWriteReadStat(t *testing.T) {
	fs := New()

	meta, err := fs.Write("/workspace/a.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if meta.Size != 5 {
		t.Fatalf("expected size 5, got %d", meta.Size)
	}

	data, stat, err := fs.Read("/workspace/a.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected hello, got %q", string(data))
	}
	if stat.Path != "/workspace/a.txt" {
		t.Fatalf("unexpected stat path %q", stat.Path)
	}
}

func TestOverwriteReplacesContent(t *testing.T) {
	fs := New()

	if _, err := fs.Write("/workspace/a.txt", []byte("hello")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	second, err := fs.Write("/workspace/a.txt", []byte("world"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if second.Size != 5 {
		t.Fatalf("expected size 5, got %d", second.Size)
	}

	data, _, err := fs.Read("/workspace/a.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(data) != "world" {
		t.Fatalf("expected world, got %q", string(data))
	}
}

func TestListAndDelete(t *testing.T) {
	fs := New()

	if _, err := fs.Write("/workspace/a.txt", []byte("a")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := fs.Write("/workspace/b.txt", []byte("b")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := fs.Write("/tmp/c.txt", []byte("c")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	files, err := fs.List("/workspace")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 workspace files, got %d", len(files))
	}

	if err := fs.Delete("/workspace/a.txt"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, _, err := fs.Read("/workspace/a.txt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

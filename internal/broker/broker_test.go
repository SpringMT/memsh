package broker

import (
	"context"
	"testing"

	"github.com/SpringMT/memsh/memfs"
)

func TestExecuteDSL(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/app.log", []byte("INFO boot\nERROR b\nERROR a\nERROR a\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	b := New(fs)
	result, err := b.ExecuteDSL(context.Background(), `grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt`)
	if err != nil {
		t.Fatalf("ExecuteDSL() error = %v", err)
	}

	if result.OutputPath != "/output/errors.txt" {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
	if string(result.Output) != "ERROR a\nERROR b\n" {
		t.Fatalf("unexpected output %q", string(result.Output))
	}
}

func TestRejectWriteToInputNamespace(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/app.log", []byte("ERROR a\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	b := New(fs)
	if _, err := b.ExecuteDSL(context.Background(), `grep "ERROR" /input/app.log > /input/errors.txt`); err == nil {
		t.Fatal("expected write to /input to be rejected")
	}
}

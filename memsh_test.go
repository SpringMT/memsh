package memsh

import (
	"context"
	"testing"
)

func TestSessionLoadExecuteRead(t *testing.T) {
	mgr := NewManager()
	s := mgr.Open()

	if err := s.Load([]File{
		{Path: "/input/app.log", Content: []byte("INFO boot\nERROR b\nERROR a\nERROR a\n")},
	}); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	result, err := s.Execute(context.Background(), `grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt`)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if string(result.Output) != "ERROR a\nERROR b\n" {
		t.Fatalf("unexpected output %q", string(result.Output))
	}

	input, _, err := s.Read("/input/app.log")
	if err != nil {
		t.Fatalf("Read(input) error = %v", err)
	}
	if string(input) != "INFO boot\nERROR b\nERROR a\nERROR a\n" {
		t.Fatalf("input mutated: %q", string(input))
	}
}

func TestSessionLoadOnceAndExecuteOnce(t *testing.T) {
	mgr := NewManager()
	s := mgr.Open()

	if err := s.Load([]File{{Path: "/input/app.log", Content: []byte("ERROR a\n")}}); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := s.Load([]File{{Path: "/input/other.log", Content: []byte("ERROR b\n")}}); err == nil {
		t.Fatal("expected second load to fail")
	}

	if _, err := s.Execute(context.Background(), `grep "ERROR" /input/app.log > /output/errors.txt`); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if _, err := s.Execute(context.Background(), `grep "ERROR" /input/app.log > /output/errors-2.txt`); err == nil {
		t.Fatal("expected second execute to fail")
	}
}

func TestSessionRejectsNonInputLoad(t *testing.T) {
	mgr := NewManager()
	s := mgr.Open()

	if err := s.Load([]File{{Path: "/output/a.txt", Content: []byte("bad")}}); err == nil {
		t.Fatal("expected non-input path to fail")
	}
}

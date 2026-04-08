package dsl

import "testing"

func TestParsePipeline(t *testing.T) {
	got, err := ParsePipeline(`grep "ERROR line" /workspace/app.log | sort | uniq > /workspace/out.txt`)
	if err != nil {
		t.Fatalf("ParsePipeline() error = %v", err)
	}
	if len(got.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(got.Commands))
	}
	if got.Commands[0].Name != "grep" {
		t.Fatalf("expected first command grep, got %q", got.Commands[0].Name)
	}
	if got.Commands[0].Args[0] != "ERROR line" {
		t.Fatalf("expected quoted arg to be preserved, got %q", got.Commands[0].Args[0])
	}
	if got.Redirect == nil || got.Redirect.Path != "/workspace/out.txt" {
		t.Fatalf("expected redirect to /workspace/out.txt, got %+v", got.Redirect)
	}
}

func TestCompilePipeline(t *testing.T) {
	plan, err := Compile(`grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "text.grep" {
		t.Fatalf("expected first tool text.grep, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].OutputPath != "/work/step-1" {
		t.Fatalf("expected first intermediate output path to be /work/step-1, got %q", plan.Steps[0].OutputPath)
	}
	if plan.Steps[2].OutputPath != "/output/errors.txt" {
		t.Fatalf("expected final output path to be preserved, got %q", plan.Steps[2].OutputPath)
	}
}

func TestRejectUnsupportedSyntax(t *testing.T) {
	if _, err := Compile(`grep ERROR /workspace/app.log && sort > /workspace/out.txt`); err == nil {
		t.Fatal("expected unsupported syntax error")
	}
}

func TestRejectMissingRedirect(t *testing.T) {
	if _, err := Compile(`grep ERROR /workspace/app.log | sort`); err == nil {
		t.Fatal("expected missing redirect error")
	}
}

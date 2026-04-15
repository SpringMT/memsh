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

func TestCompileJSONQuery(t *testing.T) {
	plan, err := Compile(`json.query '.entries[] | select(.label == "foo")' /input/data.json > /output/result.json`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "json.query" {
		t.Fatalf("expected tool json.query, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["query"] != `.entries[] | select(.label == "foo")` {
		t.Fatalf("unexpected query %q", plan.Steps[0].Params["query"])
	}
}

func TestCompileJSONQueryAlias(t *testing.T) {
	plan, err := Compile(`jq '.foo' /input/a.json > /output/b.json`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "json.query" {
		t.Fatalf("expected tool json.query, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["query"] != ".foo" {
		t.Fatalf("unexpected query %q", plan.Steps[0].Params["query"])
	}
}

func TestCompileJSONToText(t *testing.T) {
	plan, err := Compile(`json.query '.entries[]' /input/data.json | json.to_text > /output/result.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[1].Tool != "json.to_text" {
		t.Fatalf("expected second tool json.to_text, got %q", plan.Steps[1].Tool)
	}
}

func TestCompileTextReplace(t *testing.T) {
	plan, err := Compile(`text.replace 's/foo/bar/g' /input/data.txt > /output/result.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "text.replace" {
		t.Fatalf("expected tool text.replace, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["expr"] != "s/foo/bar/g" {
		t.Fatalf("unexpected expr %q", plan.Steps[0].Params["expr"])
	}
}

func TestCompileTextReplaceAlias(t *testing.T) {
	plan, err := Compile(`sed 's/a/b/g' /input/a.txt > /output/b.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "text.replace" {
		t.Fatalf("expected tool text.replace, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["expr"] != "s/a/b/g" {
		t.Fatalf("unexpected expr %q", plan.Steps[0].Params["expr"])
	}
}

func TestCompileTextCut(t *testing.T) {
	plan, err := Compile(`text.cut -d ',' -f 1,3 /input/data.csv > /output/result.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "text.cut" {
		t.Fatalf("expected tool text.cut, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["delimiter"] != "," || plan.Steps[0].Params["fields"] != "1,3" {
		t.Fatalf("unexpected params %+v", plan.Steps[0].Params)
	}
}

func TestCompileTextCutAlias(t *testing.T) {
	plan, err := Compile(`cut -d ',' -f 1 /input/a.txt > /output/b.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "text.cut" {
		t.Fatalf("expected tool text.cut, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["delimiter"] != "," || plan.Steps[0].Params["fields"] != "1" {
		t.Fatalf("unexpected params %+v", plan.Steps[0].Params)
	}
}

func TestCompileTextWC(t *testing.T) {
	plan, err := Compile(`text.wc -l /input/data.txt > /output/count.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "text.wc" {
		t.Fatalf("expected tool text.wc, got %q", plan.Steps[0].Tool)
	}
}

func TestCompileTextWCAlias(t *testing.T) {
	plan, err := Compile(`wc -l /input/a.txt > /output/b.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tool != "text.wc" {
		t.Fatalf("expected tool text.wc, got %q", plan.Steps[0].Tool)
	}
}

func TestCompileGrepIgnoreCase(t *testing.T) {
	plan, err := Compile(`grep -i 'error' /input/app.log > /output/result.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if plan.Steps[0].Tool != "text.grep" {
		t.Fatalf("expected text.grep, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["ignore_case"] != "true" {
		t.Fatalf("expected ignore_case=true, got %q", plan.Steps[0].Params["ignore_case"])
	}
	if plan.Steps[0].Params["pattern"] != "error" {
		t.Fatalf("expected pattern=error, got %q", plan.Steps[0].Params["pattern"])
	}
}

func TestCompileJSONToTextFlat(t *testing.T) {
	plan, err := Compile(`json.to_text --flat /input/data.json > /output/result.txt`)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if plan.Steps[0].Tool != "json.to_text" {
		t.Fatalf("expected json.to_text, got %q", plan.Steps[0].Tool)
	}
	if plan.Steps[0].Params["flat"] != "true" {
		t.Fatalf("expected flat=true, got %q", plan.Steps[0].Params["flat"])
	}
}

func TestRejectUnsupportedOptions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target string
	}{
		{
			name:   "grep",
			input:  `grep -v pattern /input/a.txt > /output/b.txt`,
			target: `grep: unsupported option "-v" (supported: -i)`,
		},
		{
			name:   "wc",
			input:  `wc -c /input/a.txt > /output/b.txt`,
			target: `wc: unsupported option "-c" (supported: -l)`,
		},
		{
			name:   "head",
			input:  `head -f /input/a.txt > /output/b.txt`,
			target: `head: unsupported option "-f" (supported: -n)`,
		},
		{
			name:   "cut",
			input:  `cut -z -d ',' -f 1 > /output/b.txt`,
			target: `cut: unsupported option "-z" (supported: -d, -f)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.input)
			if err == nil {
				t.Fatal("expected error")
			}
			protoErr, ok := err.(*protocolError)
			if !ok {
				t.Fatalf("expected protocolError, got %T", err)
			}
			if protoErr.Code != "unsupported_option" {
				t.Fatalf("expected unsupported_option, got %q", protoErr.Code)
			}
			if protoErr.Message != tt.target {
				t.Fatalf("expected %q, got %q", tt.target, protoErr.Message)
			}
		})
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

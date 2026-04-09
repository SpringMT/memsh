package tool

import (
	"context"
	"testing"

	"github.com/SpringMT/memsh/internal/dsl"
	"github.com/SpringMT/memsh/memfs"
)

func TestExecuteStepGrepSortUniq(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/app.log", []byte("INFO boot\nERROR b\nERROR a\nERROR a\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	steps := []dsl.Step{
		{Tool: "text.grep", InputPath: "/input/app.log", OutputPath: "/work/1", Params: map[string]string{"pattern": "ERROR"}},
		{Tool: "text.sort_lines", InputPath: "/work/1", OutputPath: "/work/2"},
		{Tool: "text.uniq_lines", InputPath: "/work/2", OutputPath: "/output/out.txt"},
	}

	for _, step := range steps {
		if err := runner.ExecuteStep(context.Background(), step); err != nil {
			t.Fatalf("ExecuteStep(%s) error = %v", step.Tool, err)
		}
	}

	got, _, err := fs.Read("/output/out.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := "ERROR a\nERROR b\n"
	if string(got) != want {
		t.Fatalf("unexpected output:\nwant %q\ngot  %q", want, string(got))
	}
}

func TestExecuteStepJSONQuery(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/data.json", []byte(`{"entries":[{"label":"foo","value":1},{"label":"bar","value":2}]}`)); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	step := dsl.Step{
		Tool:       "json.query",
		InputPath:  "/input/data.json",
		OutputPath: "/output/result.json",
		Params: map[string]string{
			"query": `.entries[] | select(.label == "foo")`,
		},
	}

	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep(%s) error = %v", step.Tool, err)
	}

	got, _, err := fs.Read("/output/result.json")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := "{\"label\":\"foo\",\"value\":1}\n"
	if string(got) != want {
		t.Fatalf("unexpected output:\nwant %q\ngot  %q", want, string(got))
	}
}

func TestExecuteStepJSONToText(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/result.json", []byte("{\"label\":\"foo\",\"value\":1}\n{\"label\":\"bar\",\"value\":2}\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	step := dsl.Step{
		Tool:       "json.to_text",
		InputPath:  "/input/result.json",
		OutputPath: "/output/result.txt",
	}

	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep(%s) error = %v", step.Tool, err)
	}

	got, _, err := fs.Read("/output/result.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := "{\n  \"label\": \"foo\",\n  \"value\": 1\n}\n{\n  \"label\": \"bar\",\n  \"value\": 2\n}\n"
	if string(got) != want {
		t.Fatalf("unexpected output:\nwant %q\ngot  %q", want, string(got))
	}
}

func TestExecuteStepTextReplace(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/data.txt", []byte("foo 123 foo\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	step := dsl.Step{
		Tool:       "text.replace",
		InputPath:  "/input/data.txt",
		OutputPath: "/output/result.txt",
		Params: map[string]string{
			"expr": `s/foo/bar/g`,
		},
	}

	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep(%s) error = %v", step.Tool, err)
	}

	got, _, err := fs.Read("/output/result.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := "bar 123 bar\n"
	if string(got) != want {
		t.Fatalf("unexpected output:\nwant %q\ngot  %q", want, string(got))
	}
}

func TestExecuteStepTextReplaceFirstOnly(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/data.txt", []byte("foo foo\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	step := dsl.Step{
		Tool:       "text.replace",
		InputPath:  "/input/data.txt",
		OutputPath: "/output/result.txt",
		Params: map[string]string{
			"expr": `s/foo/bar/`,
		},
	}

	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep(%s) error = %v", step.Tool, err)
	}

	got, _, err := fs.Read("/output/result.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := "bar foo\n"
	if string(got) != want {
		t.Fatalf("unexpected output:\nwant %q\ngot  %q", want, string(got))
	}
}

func TestExecuteStepTextCut(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/data.csv", []byte("a,b,c\n1,2,3\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	step := dsl.Step{
		Tool:       "text.cut",
		InputPath:  "/input/data.csv",
		OutputPath: "/output/result.txt",
		Params: map[string]string{
			"delimiter": ",",
			"fields":    "1,3",
		},
	}

	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep(%s) error = %v", step.Tool, err)
	}

	got, _, err := fs.Read("/output/result.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := "a,c\n1,3\n"
	if string(got) != want {
		t.Fatalf("unexpected output:\nwant %q\ngot  %q", want, string(got))
	}
}

func TestExecuteStepTextWC(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/data.txt", []byte("a\nb\nc\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	step := dsl.Step{
		Tool:       "text.wc",
		InputPath:  "/input/data.txt",
		OutputPath: "/output/count.txt",
		Params: map[string]string{
			"mode": "lines",
		},
	}

	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep(%s) error = %v", step.Tool, err)
	}

	got, _, err := fs.Read("/output/count.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := "3\n"
	if string(got) != want {
		t.Fatalf("unexpected output:\nwant %q\ngot  %q", want, string(got))
	}
}

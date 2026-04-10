package tool

import (
	"context"
	"strings"
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

func TestExecuteStepGrepRegexp(t *testing.T) {
	fs := memfs.New()
	if _, err := fs.Write("/input/app.log", []byte("INFO boot\nERROR foo\nerror bar\nWARN baz\n")); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)

	// 正規表現パターン
	step := dsl.Step{
		Tool:       "text.grep",
		InputPath:  "/input/app.log",
		OutputPath: "/output/out.txt",
		Params:     map[string]string{"pattern": "^ERROR"},
	}
	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep error = %v", err)
	}
	got, _, _ := fs.Read("/output/out.txt")
	if string(got) != "ERROR foo\n" {
		t.Fatalf("regexp: want %q, got %q", "ERROR foo\n", string(got))
	}

	// -i (ignore_case) フラグ
	step2 := dsl.Step{
		Tool:       "text.grep",
		InputPath:  "/input/app.log",
		OutputPath: "/output/out2.txt",
		Params:     map[string]string{"pattern": "error", "ignore_case": "true"},
	}
	if err := runner.ExecuteStep(context.Background(), step2); err != nil {
		t.Fatalf("ExecuteStep error = %v", err)
	}
	got2, _, _ := fs.Read("/output/out2.txt")
	if string(got2) != "ERROR foo\nerror bar\n" {
		t.Fatalf("ignore_case: want %q, got %q", "ERROR foo\nerror bar\n", string(got2))
	}
}

func TestExecuteStepJSONToTextFlat(t *testing.T) {
	fs := memfs.New()
	input := `{"label":"ナレッジ管理","description":"営業ナレッジを活用したい","belongs_to":{"map":["AIプロダクト提案"],"proposal_flow":["課題"]},"contents":{"url":"https://example.com"}}` + "\n"
	if _, err := fs.Write("/input/data.json", []byte(input)); err != nil {
		t.Fatalf("seed write error = %v", err)
	}

	runner := NewRunner(fs)
	step := dsl.Step{
		Tool:       "json.to_text",
		InputPath:  "/input/data.json",
		OutputPath: "/output/result.txt",
		Params:     map[string]string{"flat": "true"},
	}
	if err := runner.ExecuteStep(context.Background(), step); err != nil {
		t.Fatalf("ExecuteStep error = %v", err)
	}

	got, _, err := fs.Read("/output/result.txt")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	// 1行に収まっていることを確認
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("flat output should be 1 line, got %d lines:\n%s", len(lines), string(got))
	}
	// label と description が含まれることを確認
	if !strings.Contains(lines[0], "ナレッジ管理") {
		t.Fatalf("flat output should contain label, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "営業ナレッジを活用したい") {
		t.Fatalf("flat output should contain description, got %q", lines[0])
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

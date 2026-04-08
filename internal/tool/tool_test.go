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

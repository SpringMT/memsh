package broker

import (
	"context"
	"fmt"

	"github.com/SpringMT/memsh/internal/dsl"
	"github.com/SpringMT/memsh/internal/tool"
	"github.com/SpringMT/memsh/memfs"
)

type Broker struct {
	fs     *memfs.FS
	runner *tool.Runner
}

type Result struct {
	OutputPath string
	Output     []byte
}

func New(fs *memfs.FS) *Broker {
	return &Broker{
		fs:     fs,
		runner: tool.NewRunner(fs),
	}
}

func (b *Broker) ExecutePlan(ctx context.Context, plan dsl.Plan) (Result, error) {
	if len(plan.Steps) == 0 {
		return Result{}, fmt.Errorf("empty plan")
	}

	for i, step := range plan.Steps {
		if err := validateStep(step, i == len(plan.Steps)-1); err != nil {
			return Result{}, err
		}
		if err := b.runner.ExecuteStep(ctx, step); err != nil {
			return Result{}, err
		}
	}

	last := plan.Steps[len(plan.Steps)-1]
	output, _, err := b.fs.Read(last.OutputPath)
	if err != nil {
		return Result{}, fmt.Errorf("read final output %s: %w", last.OutputPath, err)
	}

	return Result{
		OutputPath: last.OutputPath,
		Output:     output,
	}, nil
}

func (b *Broker) ExecuteDSL(ctx context.Context, input string) (Result, error) {
	plan, err := dsl.Compile(input)
	if err != nil {
		return Result{}, err
	}
	return b.ExecutePlan(ctx, plan)
}

func validateStep(step dsl.Step, final bool) error {
	if !isReadablePath(step.InputPath) {
		return fmt.Errorf("input path must be under /input, /work, or /output: %q", step.InputPath)
	}
	if !isWritablePath(step.OutputPath) {
		return fmt.Errorf("output path must be under /work or /output: %q", step.OutputPath)
	}
	if isInputPath(step.OutputPath) {
		return fmt.Errorf("output path cannot target /input: %q", step.OutputPath)
	}
	if final && !isOutputPath(step.OutputPath) {
		return fmt.Errorf("final output path must be under /output: %q", step.OutputPath)
	}
	if !final && !isWorkPath(step.OutputPath) {
		return fmt.Errorf("intermediate output path must be under /work: %q", step.OutputPath)
	}
	return nil
}

func isInputPath(p string) bool {
	return p == "/input" || len(p) > len("/input/") && p[:len("/input/")] == "/input/"
}

func isReadablePath(p string) bool {
	return p == "/input" || len(p) > len("/input/") && p[:len("/input/")] == "/input/" ||
		p == "/work" || len(p) > len("/work/") && p[:len("/work/")] == "/work/" ||
		p == "/output" || len(p) > len("/output/") && p[:len("/output/")] == "/output/"
}

func isWritablePath(p string) bool {
	return isWorkPath(p) || isOutputPath(p)
}

func isWorkPath(p string) bool {
	return p == "/work" || len(p) > len("/work/") && p[:len("/work/")] == "/work/"
}

func isOutputPath(p string) bool {
	return p == "/output" || len(p) > len("/output/") && p[:len("/output/")] == "/output/"
}

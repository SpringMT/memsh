package tool

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/SpringMT/memsh/internal/dsl"
	"github.com/SpringMT/memsh/memfs"
)

type Runner struct {
	fs *memfs.FS
}

func NewRunner(fs *memfs.FS) *Runner {
	return &Runner{fs: fs}
}

func (r *Runner) ExecuteStep(ctx context.Context, step dsl.Step) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	input, _, err := r.fs.Read(step.InputPath)
	if err != nil {
		return fmt.Errorf("read input %s: %w", step.InputPath, err)
	}

	var output []byte
	switch step.Tool {
	case "fs.cat":
		output = input
	case "text.grep":
		output = grepLines(input, step.Params["pattern"])
	case "text.sort_lines":
		output = sortLines(input)
	case "text.uniq_lines":
		output = uniqLines(input)
	case "text.head":
		n, err := parseCount(step.Params, 10)
		if err != nil {
			return err
		}
		output = headLines(input, n)
	case "text.tail":
		n, err := parseCount(step.Params, 10)
		if err != nil {
			return err
		}
		output = tailLines(input, n)
	default:
		return fmt.Errorf("unknown tool %q", step.Tool)
	}

	if _, err := r.fs.Write(step.OutputPath, output); err != nil {
		return fmt.Errorf("write output %s: %w", step.OutputPath, err)
	}
	return nil
}

func parseCount(params map[string]string, fallback int) (int, error) {
	if params == nil || params["n"] == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(params["n"])
	if err != nil {
		return 0, fmt.Errorf("invalid n: %w", err)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid n: must be non-negative")
	}
	return n, nil
}

func grepLines(input []byte, pattern string) []byte {
	if pattern == "" {
		return nil
	}
	lines, endedWithNewline := splitLines(input)
	matches := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, pattern) {
			matches = append(matches, line)
		}
	}
	return joinLines(matches, endedWithNewline && len(matches) > 0)
}

func sortLines(input []byte) []byte {
	lines, endedWithNewline := splitLines(input)
	sort.Strings(lines)
	return joinLines(lines, endedWithNewline && len(lines) > 0)
}

func uniqLines(input []byte) []byte {
	lines, endedWithNewline := splitLines(input)
	if len(lines) == 0 {
		return nil
	}

	out := make([]string, 0, len(lines))
	last := lines[0]
	out = append(out, last)
	for _, line := range lines[1:] {
		if line == last {
			continue
		}
		out = append(out, line)
		last = line
	}
	return joinLines(out, endedWithNewline)
}

func headLines(input []byte, n int) []byte {
	lines, endedWithNewline := splitLines(input)
	if n < len(lines) {
		lines = lines[:n]
	}
	return joinLines(lines, endedWithNewline && len(lines) > 0)
}

func tailLines(input []byte, n int) []byte {
	lines, endedWithNewline := splitLines(input)
	if n < len(lines) {
		lines = lines[len(lines)-n:]
	}
	return joinLines(lines, endedWithNewline && len(lines) > 0)
}

func splitLines(input []byte) ([]string, bool) {
	if len(input) == 0 {
		return nil, false
	}
	endedWithNewline := bytes.HasSuffix(input, []byte("\n"))
	raw := strings.Split(strings.TrimSuffix(string(input), "\n"), "\n")
	if len(raw) == 1 && raw[0] == "" {
		return nil, endedWithNewline
	}
	return raw, endedWithNewline
}

func joinLines(lines []string, withTrailingNewline bool) []byte {
	if len(lines) == 0 {
		return nil
	}
	out := strings.Join(lines, "\n")
	if withTrailingNewline {
		out += "\n"
	}
	return []byte(out)
}

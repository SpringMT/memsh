package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/itchyny/gojq"

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
	case "json.query":
		output, err = queryJSON(input, step.Params["query"])
		if err != nil {
			return err
		}
	case "json.to_text":
		output, err = jsonToText(input)
		if err != nil {
			return err
		}
	case "text.grep":
		output = grepLines(input, step.Params["pattern"])
	case "text.replace":
		output, err = replaceText(input, step.Params["expr"])
		if err != nil {
			return err
		}
	case "text.cut":
		output, err = cutText(input, step.Params["delimiter"], step.Params["fields"])
		if err != nil {
			return err
		}
	case "text.wc":
		output, err = countText(input, step.Params["mode"])
		if err != nil {
			return err
		}
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

func queryJSON(input []byte, queryText string) ([]byte, error) {
	if queryText == "" {
		return nil, fmt.Errorf("json.query requires a query")
	}

	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return nil, fmt.Errorf("invalid json input: %w", err)
	}

	query, err := gojq.Parse(queryText)
	if err != nil {
		return nil, fmt.Errorf("invalid json query: %w", err)
	}

	iter := query.Run(value)
	results := make([][]byte, 0, 1)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("json query failed: %w", err)
		}
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("encode json query result: %w", err)
		}
		results = append(results, encoded)
	}

	if len(results) == 0 {
		return nil, nil
	}
	return append(bytes.Join(results, []byte("\n")), '\n'), nil
}

func jsonToText(input []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return nil, nil
	}

	dec := json.NewDecoder(bytes.NewReader(input))
	results := make([][]byte, 0, 1)
	for {
		var value any
		if err := dec.Decode(&value); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("invalid json input: %w", err)
		}
		pretty, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("format json output: %w", err)
		}
		results = append(results, pretty)
	}

	if len(results) == 0 {
		return nil, nil
	}
	return append(bytes.Join(results, []byte("\n")), '\n'), nil
}

func replaceText(input []byte, expr string) ([]byte, error) {
	pattern, replacement, global, err := parseReplaceExpr(expr)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid replace pattern: %w", err)
	}

	if global {
		return re.ReplaceAll(input, []byte(replacement)), nil
	}

	match := re.FindSubmatchIndex(input)
	if match == nil {
		return append([]byte(nil), input...), nil
	}

	out := make([]byte, 0, len(input))
	out = append(out, input[:match[0]]...)
	out = re.Expand(out, []byte(replacement), input, match)
	out = append(out, input[match[1]:]...)
	return out, nil
}

func cutText(input []byte, delimiter, fields string) ([]byte, error) {
	if delimiter == "" {
		return nil, fmt.Errorf("text.cut requires a delimiter")
	}

	indexes, err := parseFieldList(fields)
	if err != nil {
		return nil, err
	}

	lines, endedWithNewline := splitLines(input)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, delimiter)
		selected := make([]string, 0, len(indexes))
		for _, idx := range indexes {
			if idx-1 < len(parts) {
				selected = append(selected, parts[idx-1])
			}
		}
		out = append(out, strings.Join(selected, delimiter))
	}
	return joinLines(out, endedWithNewline && len(out) > 0), nil
}

func countText(input []byte, mode string) ([]byte, error) {
	switch mode {
	case "", "lines":
		lines, _ := splitLines(input)
		return []byte(strconv.Itoa(len(lines)) + "\n"), nil
	default:
		return nil, fmt.Errorf("unsupported wc mode %q", mode)
	}
}

func parseReplaceExpr(expr string) (pattern string, replacement string, global bool, err error) {
	if len(expr) < 2 || expr[0] != 's' {
		return "", "", false, fmt.Errorf("invalid replace expression: must start with s")
	}

	delim := expr[1]
	if delim == '\\' {
		return "", "", false, fmt.Errorf("invalid replace expression: delimiter cannot be backslash")
	}

	pattern, next, err := readDelimitedPart(expr, 2, delim)
	if err != nil {
		return "", "", false, err
	}
	replacement, next, err = readDelimitedPart(expr, next, delim)
	if err != nil {
		return "", "", false, err
	}

	flags := expr[next:]
	switch flags {
	case "":
		return pattern, replacement, false, nil
	case "g":
		return pattern, replacement, true, nil
	default:
		return "", "", false, fmt.Errorf("invalid replace flags: %q", flags)
	}
}

func readDelimitedPart(expr string, start int, delim byte) (string, int, error) {
	var b strings.Builder
	escaped := false
	for i := start; i < len(expr); i++ {
		ch := expr[i]
		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == delim {
			return b.String(), i + 1, nil
		}
		b.WriteByte(ch)
	}
	return "", 0, fmt.Errorf("invalid replace expression: unterminated expression")
}

func parseFieldList(fields string) ([]int, error) {
	if fields == "" {
		return nil, fmt.Errorf("text.cut requires at least one field")
	}

	parts := strings.Split(fields, ",")
	indexes := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid cut field %q", part)
		}
		indexes = append(indexes, n)
	}
	return indexes, nil
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

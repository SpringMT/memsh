package dsl

import (
	"fmt"
	"path"
	"strconv"
)

func Compile(input string) (Plan, error) {
	pipeline, err := ParsePipeline(input)
	if err != nil {
		return Plan{}, err
	}
	return CompilePipeline(pipeline)
}

func CompilePipeline(pipeline Pipeline) (Plan, error) {
	if pipeline.Redirect == nil {
		return Plan{}, newProtocolError("missing_output_redirect", "final output redirect is required")
	}

	outputPath, err := normalizePath(pipeline.Redirect.Path)
	if err != nil {
		return Plan{}, err
	}

	steps := make([]Step, 0, len(pipeline.Commands))
	var stdinPath string

	for i, cmd := range pipeline.Commands {
		step, nextOutput, err := compileCommand(cmd, stdinPath, stageOutputPath(i, len(pipeline.Commands), outputPath))
		if err != nil {
			return Plan{}, err
		}
		steps = append(steps, step)
		stdinPath = nextOutput
	}

	return Plan{Steps: steps}, nil
}

func stageOutputPath(index, total int, finalOutput string) string {
	if index == total-1 {
		return finalOutput
	}
	return fmt.Sprintf("/work/step-%d", index+1)
}

func compileCommand(cmd Command, stdinPath, outputPath string) (Step, string, error) {
	switch cmd.Name {
	case "cat":
		if len(cmd.Args) != 1 {
			return Step{}, "", newProtocolError("missing_argument", "cat requires exactly one path argument")
		}
		inputPath, err := normalizePath(cmd.Args[0])
		if err != nil {
			return Step{}, "", err
		}
		return Step{
			Tool:       "fs.cat",
			InputPath:  inputPath,
			OutputPath: outputPath,
		}, outputPath, nil
	case "grep":
		return compileGrep(cmd, stdinPath, outputPath)
	case "sort":
		return compileSingleInputTool(cmd, stdinPath, outputPath, "text.sort_lines")
	case "uniq":
		return compileSingleInputTool(cmd, stdinPath, outputPath, "text.uniq_lines")
	case "head":
		return compileHeadTail(cmd, stdinPath, outputPath, "text.head")
	case "tail":
		return compileHeadTail(cmd, stdinPath, outputPath, "text.tail")
	default:
		return Step{}, "", newProtocolError("unknown_command", fmt.Sprintf("unknown command %q", cmd.Name))
	}
}

func compileGrep(cmd Command, stdinPath, outputPath string) (Step, string, error) {
	if len(cmd.Args) < 1 || len(cmd.Args) > 2 {
		return Step{}, "", newProtocolError("missing_argument", "grep requires a pattern and optional path")
	}
	step := Step{
		Tool:       "text.grep",
		OutputPath: outputPath,
		Params: map[string]string{
			"pattern": cmd.Args[0],
		},
	}
	if len(cmd.Args) == 2 {
		inputPath, err := normalizePath(cmd.Args[1])
		if err != nil {
			return Step{}, "", err
		}
		step.InputPath = inputPath
		return step, outputPath, nil
	}
	if stdinPath == "" {
		return Step{}, "", newProtocolError("missing_pipeline_input", "grep requires pipeline input or an explicit path")
	}
	step.InputPath = stdinPath
	return step, outputPath, nil
}

func compileSingleInputTool(cmd Command, stdinPath, outputPath, tool string) (Step, string, error) {
	if len(cmd.Args) > 1 {
		return Step{}, "", newProtocolError("invalid_argument", fmt.Sprintf("%s accepts at most one path argument", cmd.Name))
	}
	step := Step{
		Tool:       tool,
		OutputPath: outputPath,
	}
	switch len(cmd.Args) {
	case 1:
		inputPath, err := normalizePath(cmd.Args[0])
		if err != nil {
			return Step{}, "", err
		}
		step.InputPath = inputPath
	case 0:
		if stdinPath == "" {
			return Step{}, "", newProtocolError("missing_pipeline_input", fmt.Sprintf("%s requires pipeline input or an explicit path", cmd.Name))
		}
		step.InputPath = stdinPath
	}
	return step, outputPath, nil
}

func compileHeadTail(cmd Command, stdinPath, outputPath, tool string) (Step, string, error) {
	step := Step{
		Tool:       tool,
		OutputPath: outputPath,
		Params:     map[string]string{},
	}

	args := cmd.Args
	if len(args) >= 2 && args[0] == "-n" {
		if _, err := strconv.Atoi(args[1]); err != nil {
			return Step{}, "", newProtocolError("invalid_argument", fmt.Sprintf("%s requires an integer after -n", cmd.Name))
		}
		step.Params["n"] = args[1]
		args = args[2:]
	}

	if len(args) > 1 {
		return Step{}, "", newProtocolError("invalid_argument", fmt.Sprintf("%s accepts at most one path argument", cmd.Name))
	}
	if len(args) == 1 {
		inputPath, err := normalizePath(args[0])
		if err != nil {
			return Step{}, "", err
		}
		step.InputPath = inputPath
		return step, outputPath, nil
	}
	if stdinPath == "" {
		return Step{}, "", newProtocolError("missing_pipeline_input", fmt.Sprintf("%s requires pipeline input or an explicit path", cmd.Name))
	}
	step.InputPath = stdinPath
	return step, outputPath, nil
}

func normalizePath(raw string) (string, error) {
	if raw == "" {
		return "", newProtocolError("invalid_path", "path cannot be empty")
	}
	if raw[0] != '/' {
		return "", newProtocolError("invalid_path", fmt.Sprintf("path must be absolute: %q", raw))
	}
	clean := path.Clean(raw)
	if clean == "." || clean == "/" && raw != "/" {
		return clean, nil
	}
	if clean == ".." || len(clean) < 1 || clean[0] != '/' {
		return "", newProtocolError("invalid_path", fmt.Sprintf("invalid path: %q", raw))
	}
	return clean, nil
}

func newProtocolError(code, message string) error {
	return &protocolError{Code: code, Message: message}
}

type protocolError struct {
	Code    string
	Message string
}

func (e *protocolError) Error() string {
	return e.Code + ": " + e.Message
}

# memsh

`memsh` is a lightweight, memory-first broker for LLM-driven data operations.

It lets an LLM use a restricted shell-like DSL such as:

```sh
grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt
```

without ever executing that text as a real shell command.

Instead, `memsh`:

1. parses the DSL with a restricted grammar
2. validates the input against a safe command set
3. compiles the pipeline into a structured execution plan
4. executes built-in tools against in-memory data

## Why

Projects like `just-bash` or agent-style shell bridges are useful, but they often rely on real shell execution as the control surface.

`memsh` takes a different path:

- no arbitrary shell execution
- no subprocesses in v1
- memory-only by default
- session-scoped in-memory isolation
- immutable input data after load
- built-in tools instead of wrapped commands
- explicit, auditable execution plans

The target use case is an OSS-friendly runtime for giving LLMs a shell-like mental model while keeping the actual execution surface narrow and defensible.

## Current Status

This repository currently includes:

- a v1 DSL specification
- a minimal in-memory filesystem
- a session manager with immutable `/input` loading
- tests for syntax acceptance and rejection
- small demo and example programs

Not implemented yet:

- richer built-in tool coverage beyond the current JSON and text primitives
- public API refinement

## DSL Example

Input:

```sh
grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt
```

Compiled plan:

```json
{
  "steps": [
    {
      "tool": "text.grep",
      "inputPath": "/input/app.log",
      "outputPath": "/work/step-1",
      "params": {
        "pattern": "ERROR"
      }
    },
    {
      "tool": "text.sort_lines",
      "inputPath": "/work/step-1",
      "outputPath": "/work/step-2"
    },
    {
      "tool": "text.uniq_lines",
      "inputPath": "/work/step-2",
      "outputPath": "/output/errors.txt"
    }
  ]
}
```

## Public Packages

- `memsh`: the public API for isolated execution sessions

The rest of the implementation lives under `internal/` so the execution model can evolve without expanding the public contract.

## Go API

Typical usage looks like this:

```go
mgr := memsh.NewManager()
sess := mgr.Open()
defer mgr.Close(sess.ID())

err := sess.Load([]memsh.File{
	{Path: "/input/app.log", Content: data},
})
if err != nil {
	return err
}

result, err := sess.Execute(ctx, `grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt`)
if err != nil {
	log.Printf("exit=%d stderr=%s", result.ExitCode, string(result.Stderr))
	return err
}

fmt.Println(string(result.Output))
```

## Examples

Use the in-memory filesystem:

```sh
go run ./examples/memfs
```

Execute a DSL pipeline against MemFS:

```sh
go run ./examples/execute
```

Query JSON skill data and format it for LLM output:

```sh
go run ./examples/skillquery
```

## Session Model

`memsh` is designed for short-lived sessions created and owned by the caller.

- a session is opened
- input files are loaded under `/input/**`
- loaded input becomes immutable
- one or more DSL executions produce `/work/**` intermediates and `/output/**` results
- later executions may read previously written `/work/**` and `/output/**`
- the session is closed and discarded

## Development

```sh
GOCACHE=$(pwd)/.gocache go test ./...
```

## Spec

The current v1 DSL spec lives at [docs/v1-spec.md](docs/v1-spec.md).

## Current Commands

- `cat`
- `json.query`
- `json.to_text`
- `text.replace`
- `text.cut`
- `text.wc`
- `grep`
- `sort`
- `uniq`
- `head`
- `tail`

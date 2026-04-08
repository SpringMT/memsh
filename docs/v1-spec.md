# memsh v1 DSL Spec

## Goal

`memsh` is a lightweight broker for LLM-driven data operations that feel shell-like without granting direct shell access.

The design goal is:

- lightweight
- memory-only by default
- safe by construction
- easy for LLMs to emit

In v1, the system does **not** execute arbitrary shell commands.
Instead, it accepts a restricted shell-like DSL, parses it into a safe AST, validates it, and compiles it into built-in capability calls executed against an in-memory filesystem.

## Model

There are three layers:

1. LLM-facing DSL
2. Broker AST and execution plan
3. Built-in tools operating on MemFS

The LLM may produce input such as:

```sh
grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt
```

The broker must:

1. parse the text with a restricted grammar
2. reject any unsupported syntax
3. compile the AST into a plan of safe built-in tool invocations
4. execute the plan only against MemFS paths

## Security Principles

- No direct shell execution
- No subprocess execution in v1
- No variable expansion
- No command substitution
- No globbing
- No filesystem access outside MemFS
- Immutable inputs after session load
- No implicit writes
- Every write target is explicit
- Every compiled step must map to a built-in tool

## DSL Scope

### Allowed

- command pipelines with `|`
- stdout redirection with `>`
- quoted strings with `"` or `'`
- plain words

### Rejected

- `;`
- `&&`
- `||`
- `<`
- `>>`
- `2>`
- backticks
- `$VAR`
- `$(...)`
- `*`, `?`, `[]` globbing
- subshells

## Grammar

Approximate grammar:

```bnf
pipeline   := command ( "|" command )* redirect?
redirect   := ">" path
command    := word argument*
argument   := word
word       := bareword | singlequoted | doublequoted
path       := word
```

Additional lexical rules:

- whitespace separates words
- quotes preserve whitespace
- backslash escaping is intentionally minimal in v1
- empty pipeline stages are invalid

## Built-in Commands

These names are accepted in the DSL and compiled into built-in tools.

- `cat`
- `grep`
- `sort`
- `uniq`
- `head`
- `tail`

These are not shell commands at runtime. They are only DSL command names.

## Command Semantics

### `cat`

- `cat <path>`
- reads a MemFS file and forwards its content to the next stage

### `grep`

- `grep <pattern> [path]`
- if `path` is present, reads from the given path
- otherwise reads from pipeline stdin

### `sort`

- `sort [path]`
- sorts input by lines

### `uniq`

- `uniq [path]`
- removes adjacent duplicate lines

### `head`

- `head [-n N] [path]`

### `tail`

- `tail [-n N] [path]`

## Path Rules

- Paths are logical MemFS paths, not OS paths
- Paths must be absolute and normalized
- Paths must start with `/`
- `.` and `..` are normalized away and may not escape root
- Input files live under `/input`
- Intermediate compiler outputs live under `/work`
- Final user-visible outputs live under `/output`
- `/input` is immutable after the session is loaded

## Compilation Model

The broker compiles each pipeline stage into one plan step.

Example input:

```sh
grep "ERROR" /input/app.log | sort | uniq > /output/errors.txt
```

Compiles to:

1. `text.grep(input=/input/app.log, pattern="ERROR", output=/work/step-1)`
2. `text.sort_lines(input=/work/step-1, output=/work/step-2)`
3. `text.uniq_lines(input=/work/step-2, output=/output/errors.txt)`

If the last stage has no explicit `>` target, compilation fails in v1.
This avoids implicit stdout handling in the first version.

## Broker Validation Rules

The broker must reject input when:

- a command name is unknown
- unsupported syntax is present
- a command requires pipeline input but none exists
- a path argument is missing when required
- the final output path is missing
- option syntax is unsupported
- too many arguments are provided for a command
- a step tries to write under `/input`
- a final redirect does not target `/output`

## Session Model

`memsh` is designed for short-lived, one-shot sessions.

1. open a session
2. load immutable input files under `/input/**`
3. execute one DSL pipeline
4. read `/output/**`
5. close and discard the session

The session is not intended to be reused across multiple jobs.

## Go Library Usage

`memsh` is intended to be used as a Go library, not as a standalone server.

The expected call flow is:

1. create a `memsh.Manager`
2. open a session
3. load immutable input files under `/input/**`
4. execute DSL through the session
5. read `/output/**` or use the returned output bytes
6. close the session

The current implementation exposes a small public surface centered on `memsh`. Parsing, planning, and tool execution live under `internal/`.

## Error Codes

Suggested error codes:

- `invalid_syntax`
- `unsupported_syntax`
- `unknown_command`
- `missing_argument`
- `invalid_argument`
- `missing_pipeline_input`
- `missing_output_redirect`
- `invalid_path`
- `invalid_namespace`

## Why This Design

This design keeps the LLM experience shell-like enough to be usable, while ensuring execution remains constrained, auditable, and memory-only.

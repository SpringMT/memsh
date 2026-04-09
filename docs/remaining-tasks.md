# Remaining Tasks

This document tracks the follow-up work after the current JSON/text feature expansion in `memsh`.

## High Priority

- Add end-to-end session tests for multi-step pipelines such as `json.query | json.to_text` and `text.cut | text.wc`.
- Expand structured execution errors beyond `ExitCode` and `Stderr` to include machine-readable fields such as error code, failing step index, and tool name.
- Review and update the public docs so they fully match the current behavior, especially the multi-execute session model and JSON-oriented commands.

## Medium Priority

- Add an example closer to `run_skill_script`, where arguments are translated into a memsh DSL command before execution.
- Clarify command limitations in the docs:
  - `json.query` uses `gojq` but does not guarantee full `jq` compatibility.
  - `text.replace` currently supports only `s/pattern/replacement/flags` with the optional `g` flag.
  - `text.cut` currently supports only comma-separated 1-based field lists such as `1,3`.
  - `text.wc` currently supports only line counting via `-l`.
- Add CI coverage for example execution, especially `go run ./examples/skillquery`.

## Low Priority

- Consider richer text processing primitives if real skill assets need them.
- Consider a versioned spec refresh if the project no longer wants to describe the runtime as the original `v1` shape.

# Specification: Format Agent Max Turns Stop Message and Allow Configurable Turn Limits

**Task Slug:** `fix-273-agent-max-turns-message`
**Date:** `2026-10-08`
**Intent:** [.agents/tasks/fix-273-agent-max-turns-message/intent.md](intent.md)

## 1. Target Architecture & Interfaces
- **Default Turns Ceiling**: Increase `DefaultTurns` in `claudecode` package and default `Claude.MaxSteps` in `agent` package from `30` to `100`.
- **Subtype Formatter (`formatSubtype`)**:
  - Unexported helper in `modules/apps/desktop/internal/adapter/claudecode/read.go` mapping machine error subtypes to domain-phrased messages.
  - Formatted messages conform to project voice (concise, lowercase, e.g., `"the agent reached the limit of steps without finishing"`).
- **Result Parsing Logic (`result`)**:
  - If `said.Result` contains meaningful prose (distinct from the subtype string itself), return `said.Result`.
  - If `said.Subtype` is non-empty, return the output of `formatSubtype(said.Subtype)`.
  - Default fallback: `"the agent stopped without finishing"`.

## 2. Blast Radius Matrix
| Package / Path | File | Action (Create/Modify) | Downstream Consumers |
| :--- | :--- | :--- | :--- |
| `modules/apps/desktop/internal/adapter/claudecode` | `read.go` | Modify | `Take()` event streaming consumer, UI thread turns |
| `modules/apps/desktop/internal/adapter/claudecode` | `claudecode.go` | Modify | `Take()` subprocess argument construction |
| `modules/apps/desktop/internal/adapter/claudecode` | `claudecode_test.go` | Modify | Test suite for Claude Code adapter |
| `modules/libs/core/adapter/agent` | `config.go` | Modify | Settings unmarshaling & defaults (`container.Config`) |

## 3. Negative Invariants (Architectural Prohibitions)
- Do not introduce new external dependencies or libraries.
- Do not modify wire protocol definitions (`modules/libs/protocol`) or DTO shapes.
- Do not alter the signature of `port.Agent` or `Take(ctx context.Context, task port.Task)`.
- No narrative comments or ADR citations in code comments (comments state the rule and stop).

## 4. Acceptance Criteria

### AC-1: Turn Limit Subtype Translation
- **Given**: An agent subprocess finishes with stream result `{"type":"result","subtype":"error_max_turns","is_error":true}`.
- **When**: The output stream is read and completed.
- **Then**: The final step is `port.StepStopped` with `Detail` equal to `"the agent reached the limit of steps without finishing"`.

### AC-2: Other Error Subtype Translation
- **Given**: An agent subprocess finishes with stream result `{"type":"result","subtype":"error_context_length_exceeded","is_error":true}`.
- **When**: The output stream is read and completed.
- **Then**: The final step is `port.StepStopped` with `Detail` equal to `"the conversation is too long for the model"`.

### AC-3: Custom Result Prose Preserved
- **Given**: An agent subprocess finishes with `{"type":"result","subtype":"error_max_turns","is_error":true,"result":"went round too many times"}`.
- **When**: The output stream is read and completed.
- **Then**: The final step is `port.StepStopped` with `Detail` equal to `"went round too many times"`.

### AC-4: Default Turn Limit Increased to 100
- **Given**: An agent configured with default options (`Turns == 0`).
- **When**: The agent is launched with `Take()`.
- **Then**: The `--max-turns` argument passed to the child process is `"100"`.

### AC-5: Custom Turn Limit Honored
- **Given**: An agent configured with custom `Turns = 50`.
- **When**: The agent is launched with `Take()`.
- **Then**: The `--max-turns` argument passed to the child process is `"50"`.

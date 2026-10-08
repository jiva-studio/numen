# Intent: Format Agent Max Turns Stop Message and Allow Configurable Turn Limits

**Task Slug:** `fix-273-agent-max-turns-message`
**Date:** `2026-10-08`

## 1. Problem & JTBD (The "Why")
- **Pain & Trigger:** When the assistant works through multi-step research, note searching, or note editing, it hits a hard ceiling of 30 steps. When it stops at this limit, the user interface displays a cryptic raw machine code string (`error_max_turns`) instead of an explanatory sentence in the application's natural voice.
- **Target User & Scenario:** A user asking complex questions or requesting extensive updates across notes that require more than 30 exploratory and execution turns.
- **Desired Outcome:** 
  1. The default turn limit is increased to a higher reasonable default (100 turns) and respects user configuration for maximum turn allowance.
  2. If the assistant reaches its turn limit, it presents a clear, polite explanation in the application voice stating that the agent reached its step limit without finishing, instead of leaking raw API error identifiers.

## 2. Scope Boundaries & Strict Non-Goals
### In Scope (Goals):
- Human-friendly translation of assistant termination and error reasons (specifically turn limit exhaustion and known error subtypes) into natural domain sentences.
- Raising the default step limit ceiling from 30 to 100 turns for more robust multi-step note interactions.
- Preserving custom step limit overrides provided in the user's settings.

### Strictly Out of Scope (Non-Goals):
- Changing the underlying agent execution model, streaming transport, or process lifecycle.
- Modifying client UI component presentation or layout of stopped messages beyond the wording received.
- Auto-resuming or silently retrying when an agent hits its turn limit.

## 3. Limits & Failure Modes
| Scenario | Expected Behavior |
| :--- | :--- |
| Assistant reaches configured max turn ceiling | Assistant stops and informs user: "the agent reached the limit of steps without finishing". |
| Assistant encounters unexpected API failure subtype | Machine error identifiers are translated to readable sentences or clear descriptions rather than raw snake_case codes. |
| User configures custom turn ceiling in settings | Assistant runs with the specified turn limit rather than the default. |
| User does not specify turn ceiling | Assistant defaults to 100 turns. |

## 4. Invariants & Business Constraints
- Invariant 1: Raw machine error codes (e.g. `error_max_turns`) must never leak directly into the user-facing chat interface.
- Invariant 2: Explicit user-configured step limits must take precedence over default values.
- Invariant 3: Stopping reasons must follow the concise, lowercase project voice ("the agent...").

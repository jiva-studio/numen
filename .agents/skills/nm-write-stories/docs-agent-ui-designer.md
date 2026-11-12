# Proposal: the `ui-designer` agent

This is a proposal. The definition below is not under `.agents/agents/`; the owner decides whether it is added, changed or dropped.

## Why

`coder` implements to a spec and `tester` writes acceptance tests, in every language. Work in `libs/ui` is different: the result is a component that is drawn, so the work is design (states, tokens, density, keyboard behavior) before it is code, and the stories are both the design surface and the tests. A general `coder` has no guide for what a story shows and no mandate over `libs/ui` alone. `ui-designer` is a `coder` specialised to `libs/ui/**` that preloads the skills for placement and stories.

## Definition

```markdown
---
name: ui-designer
description: Design and implementation agent for libs/ui/components (@numen/ui) and libs/ui/workbench (@numen/workbench). Builds a component, its stories, its unit test and its fixtures with design tokens, and returns the component with its list of states and behaviors. Writes under libs/ui only.
tools: Read, Write, Edit, Glob, Grep, Bash
skills:
  - nm-write-stories
  - nm-place-library
---

You are the **UI design agent**, a specialisation of the implementation role in `.agents/rules/process.md` for `libs/ui`.

Before writing, read:

1. `.agents/skills/nm-write-stories/SKILL.md` — what a story shows, how a `play` function drives it, fixtures, clock and viewport, focus and names, tokens.
2. `.agents/skills/nm-place-library/SKILL.md` — whether the component belongs in `@numen/ui`, `@numen/workbench` or in a window.
3. `.agents/rules/coding-style-frontend.md` and `.agents/rules/architecture.md` sections 2, 4 and 5.
4. `libs/ui/components/src/shared/tokens/tokens.css` and `theme.css` — the tokens that exist — and `docs/themes.md` for how a theme redeclares them.
5. `spec.md` in the task folder if an active task exists.

Core rules:

- **Write under `libs/ui/**` only.** A change another path needs (a window that renders the component, a plugin, the protocol) is reported, not made.
- **Decide placement first.** If the component names a domain term or needs a wire type, stop and report with the drawing vocabulary you propose; do not build it in `@numen/ui`.
- **List the states and behaviors before the first line of code**, from the props, events and slots: the ordinary content, each value that changes what is drawn, the data edge cases, and each behavior a person performs. The list is the first thing you return.
- **Build the pure logic first** as a module beside the component with its own unit test; the `.vue` file only wires.
- **Draw with tokens.** No colour, font or duration literal, no `dark:`, no palette class. Add a token to `tokens.css` and `theme.css` when two components need the value; a length is a `rem` or `em` literal on the property or a token; `px` only for a 1px hairline.
- **Write the stories.** One story per state and one `play` function per behavior, driven with `userEvent`, found by role and name, spying through `fn()` on the emitted events only. Fixtures are invented and general; none comes from a vault or a wire type.
- **Test no pixel, gap, scale or contrast ratio, and take no picture.** A story proves behavior: open, close, flip, move.
- **The clock, the viewport, randomness and identifiers are parameters** with browser defaults.
- **Do not edit acceptance tests.** If a spec exists and the `tester` wrote stories or tests for it, make them pass and report any you think is wrong; do not change them.
- **Do not weaken a check.** No `eslint-disable`, no `@ts-ignore`, no exception in a configuration.
- **Strict engineering language**: `AGENTS.md` section 13. Plain technical English in comments, story names and doc comments.
- **Run only what is light while editing**: `npm run typecheck` and `npm run lint` in the package. Do not run the story projects; the verifier runs them once at the end, one project at a time (`npm run test:unit`, `vitest run --project 'stories (chromium)'`, `vitest run --project 'stories (webkit)'`).
- **Return**: the paths written; the list of states and behaviors with the story that covers each; the tokens used and any token added; anything that the owner looks at (the accessible names chosen, the tab order, the focus destination after close); the decisions taken without a source.
- **Next step**: instruct the user to run `/nm-review`.
```

## What it may write

Everything under `libs/ui/components/**` and `libs/ui/workbench/**`: components, their `*.types.ts`, pure modules and their tests, stories, fixtures, `tokens.css` and `theme.css`, the package `index.ts` and `src/index.ts`. A frontmatter allowlist names tools, not paths, so the path limit is the system prompt's rule; a repository hook that refuses a write outside `libs/ui/**` for this agent would enforce it, and the owner decides whether to add one.

It does not write: windows, plugins, `libs/core/client`, `libs/core/protocol`, the SDKs, Go code, the `.storybook` configuration (a change there is a proposal to the owner, because it changes every story), `tools/`, `.agents/`.

## What it returns

A component folder that typechecks and lints, and the report named in the last bullet of the prompt. The report is short and ordered: placement, states and behaviors with their stories, tokens, what a person still decides, decisions without a source.

## Relation to the other roles

- `tester`. With a `spec.md`, the `tester` writes the behavior stories (the ones with a `play` function) and the unit tests red first, as `process.md` section 1 says, and `ui-designer` makes them green and writes the state stories, the pure modules and their tests. Without a spec (a new component under design), `ui-designer` writes both, and the `reviewer` is the independent check. The state stories carry no acceptance weight: they are the design surface.
- `coder`. `coder` keeps everything outside `libs/ui`. When a feature needs a new component and its window, the sequence is `ui-designer` for the component, then `coder` for the window that renders it and maps the domain onto the component's props. The two never write the same file.
- `reviewer`. `/nm-review` runs as for any change; the stages read the rules the skill states. The reviewer breaks the stories: removes the behavior and checks the story fails, feeds the component an empty list, a name with nothing to break at, a text in another script.
- The owner. Look at the story in both themes and at 200 percent of the interface scale before accepting the design; the agent returns the pointers.

## Open points

- The skills named in the `skills` frontmatter (`nm-write-stories`, `nm-place-library`) exist in `.agents/skills/`; the owner decides whether the agent is added.
- Whether a path-limiting hook is wanted.
- Whether `Bash` is needed: it runs `typecheck` and `lint`. Without it the agent cannot check its own work and the coordinator runs the checks.
- Whether the agent may add a token on its own, or only propose it. This proposal lets it add one when two components use the value, as `nm-place-library` allows.

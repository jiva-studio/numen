# The window may frame the hosts a video plays from

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`
- **Related:** [How an interface component is built](0023-how-an-interface-component-is-built.md), [A link is a note that carries an address](0038-a-link-is-a-note-that-carries-an-address.md)

## Context

The window loads what its own handler serves and nothing else. No script runs that it did not serve, no form is submitted anywhere, and until now nothing reached off the machine at all.

A person who pasted a video wants to watch it.

## Decision

### `frame-src` names the hosts, and nothing else widens

The policy takes a field of its own for where a frame may come from, filled with the embed hosts and no others. Widening where a frame comes from does not widen where a picture, a sound or a script does: one field to a directive is what that type is for.

`script-src` stays as it was. The frame is given the parameter that makes a player answer messages, and the page holding it sends those messages itself, so seeking to the second a passage was said runs none of that host's code in the window.

The frame is sandboxed, is told which page holds it, and sends nothing about where the person came from.

### The list is closed, and it is in the code

A host is framed only where there is also a way to compose the address a frame plays from, and that is written for one host at a time. A setting naming a host nothing knows how to embed would be a setting that does nothing.

## Consequences

- A window showing a link tab reaches off the machine, and the sentence saying
it never does is gone.
- A link tab plays nothing offline.
- Adding a host is a change to the policy and to what composes an embed
address, in one diff somebody reads.

## Alternatives considered

**No frame: a copy is downloaded and played locally.** Rejected as the only way: the person pasted an address to watch what is there, and an hour of video on their disk is not what they asked for. It stands as the other way, asked for by hand — a copy is played in place of the frame, from the vault's own folder, and a note with one loads nothing of the site at all.

**The host's own player script, loaded into the window.** Rejected: it would run in the window rather than inside a frame of its own, which is exactly what the policy exists to stop.

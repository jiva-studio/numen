# Downloading what is at an address

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`
- **Related:** [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [Where a port is declared and where an adapter stands](0035-where-a-port-is-declared-and-where-an-adapter-stands.md), [A url is a source of its own](0038-a-url-is-a-source-of-its-own.md)

## Context

A person keeps an address in their vault because of what is at it: a lecture, an article, a talk. What is at an address is somebody else's. It changes, it is published in whatever shape that site chose this month, and one day it is not there at all.

So the application downloads it. The words come into the vault and are searched, cut and quoted like everything else, and the video itself can come down too, so a thing worth keeping does not depend on a site still serving it.

## Decision

### The application downloads what is at an address, and the vault keeps it

Two different things come down, and a person asks for them separately.

**The words.** What a site published with a video, or the prose a page is written around. They are small, they are what makes the address searchable, and they come down when the address is imported.

**The media.** The video or the sound itself. It is asked for by hand: an hour of video on somebody's disk is not what pasting an address asks for, and how large a copy may be at all is a setting.

Either can be thrown away without touching the other, and the address still points where it pointed.

### Downloading is one port, and each site is answered behind it

The core asks in its own words: *what is at this address, what it is called, how long it runs, the words published with it, a copy of it*. One port.

Behind it, each source is answered on its own, and a source that knows a site better is asked before one that knows it less. A page is what an address is when nothing knows the site better, so pages are answered last. Teaching the application a new site is written in one place and nothing else is told about it.

### The programs that speak to sites are the machine's

Keeping up with what a site publishes and how is a full-time job, and the programs that do it are maintained by people who do nothing else and are already on the machines of the people who would use this.

So the application runs them rather than reimplementing them. Which program, and how it is started, is a setting; an empty setting asks the path. Whatever a person needs to pass a site — cookies from their browser, a token, a proxy — they pass in that setting, and it is handed to every run. What the program said when it refused is what they are shown: it knows why, and nothing here says it better.

Neither program is linked into the application and neither is shipped with it. A separate program run over a pipe is neither, so the application's licence stays its own.

### A missing program costs the addresses it would have reached, and nothing else

A page needs no program, so every machine can download the words of a page. A machine without the one that reaches videos reaches no video, and asking for one is answered by naming the program that would have. The address is what is refused, not the application.

Nothing is said about it at startup. An absent program is a build that does less, and every other part of the application that a machine may or may not carry is silent about it too.

## Consequences

- What is at an address outlives the site: the words are in the vault, and the
  media can be too.
- A machine without a program loses the addresses that program reaches, and is
  told which one it wants.
- Downloading from another site is one addition in one place.
- What a site does to keep programs out is answered by the person's own
  settings, without waiting for a release.
- Nothing here is tested against a network: what a real run said is recorded,
  and the tests read that.

## Alternatives considered

**Speaking to the sites from here.** Rejected: it is a full-time job somebody else is already doing, and a month behind it is a feature that does not work.

**Shipping the programs inside the application.** Rejected: it takes on their licences, and it means shipping a program that has to be current to work.

**Downloading the media with the words.** Rejected: pasting an address is a cheap gesture, and an hour of video is not.

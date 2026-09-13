/**
 * What the window says about a recording tab, asked without a screen.
 *
 * How far a recording has been written down and how long it runs are both
 * milliseconds, and both are what is reported of the tab.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { recordingKind, useTranscriptTab, type MediaTabState, type Medium } from './kind'
import type { TranscriptState } from './model/transcript'
import type { FileOpeners, WindowHandle } from '@/entities/tab/@x/media'

/** A medium drawn by nothing, which is as much of one as a kind is asked for. */
const played: Medium = {
  tab: 'recording',
  source: 'recording',
  pane: {},
  register: (puts, read) => puts.registerReader({ kind: played.source }, read),
}

/** A window holding no tabs. What is asked here is over the tab in hand. */
const tabs = { openTab: async () => '', getTabState: () => null } as unknown as WindowHandle

/** Openers that take the reader the kind hands over and ask nothing of it. */
const openers = { registerReader: () => {} } as unknown as FileOpeners

/** A recording open in a tab, as far as the window reads one. */
const recording = (path: string, transcript: number, duration: number) =>
  ({
    path,
    transcribedDuration: ref(transcript),
    duration: ref(duration),
  }) as unknown as MediaTabState

/** The kind, made over the one recording tab this test hands it. */
const kind = (state: MediaTabState) =>
  recordingKind(
    tabs,
    () => state as unknown as TranscriptState,
    { runCommand: () => {} },
    openers,
    played,
  ).kind

describe('what a command asked over a recording tab is over', () => {
  it('is the file it plays, which is what a run is asked over', () => {
    const held = recording('talks/Ants.mp3', 4000, 9000)

    expect(kind(held).getTarget!(held)).toStrictEqual({
      file: 'talks/Ants.mp3',
      source: 'recording',
    })
  })
})

describe('what a recording tab holds, as whoever answers for the person is told it', () => {
  it('is the file, how far it is written down, and how long it runs', () => {
    const held = recording('talks/Ants.mp3', 4000, 9000)

    expect(kind(held).getOpenTab!(held)).toStrictEqual({
      path: 'talks/Ants.mp3',
      recording: { transcribedDurationMs: 4000, durationMs: 9000 },
    })
  })
})

/**
 * A recording as the tab reads it: what it was found to hold, and whether the
 * finding has landed.
 */
const held = (how: { isLoading: boolean; times: readonly unknown[] }) =>
  ({
    path: 'talks/Ants.mp3',
    times: ref(how.times),
    points: ref(false),
    isWorking: ref(false),
    isLoading: ref(how.isLoading),
  }) as unknown as TranscriptState

describe('a recording whose words have not been read yet', () => {
  it('offers none of the runs over them', () => {
    const tab = useTranscriptTab(held({ isLoading: true, times: [] }), { runCommand: () => {} })

    expect(tab.transcribable.value).toBe(false)
    expect(tab.proofreadable.value).toBe(false)
    expect(tab.deletable.value).toBe(false)
  })

  it('offers writing them down once the reading has landed and found none', () => {
    const tab = useTranscriptTab(held({ isLoading: false, times: [] }), { runCommand: () => {} })

    expect(tab.transcribable.value).toBe(true)
  })

  it('offers putting them right once the reading has landed and found some', () => {
    const tab = useTranscriptTab(held({ isLoading: false, times: [{}] }), { runCommand: () => {} })

    expect(tab.transcribable.value).toBe(false)
    expect(tab.proofreadable.value).toBe(true)
    expect(tab.deletable.value).toBe(true)
  })
})

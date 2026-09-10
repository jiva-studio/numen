/**
 * What the window says about a recording tab, asked without a screen.
 *
 * How far a recording has been written down and how long it runs are both
 * milliseconds, and both are what is reported of the tab.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { recordingKind, type MediaTabState, type Medium } from './kind'
import type { TranscriptState } from './transcript'
import { fileOpeners } from '../tabs/openers'
import { windowTabs } from '../tabs/windowTabs'
import { RECORDING } from '../tabs/workspace'

/** A medium drawn by nothing, which is as much of one as a kind is asked for. */
const played: Medium = {
  tab: RECORDING,
  source: 'recording',
  draws: {},
  hands: (puts, opens) => puts.reads({ kind: played.source }, opens),
}

/** A recording open in a tab, as far as the window reads one. */
const recording = (path: string, transcribedDuration: number, duration: number) =>
  ({
    path,
    transcribedDuration: ref(transcribedDuration),
    duration: ref(duration),
  }) as unknown as MediaTabState

/** The kind, made with a window that opens recordings this test hands it. */
const kind = (held: MediaTabState) => {
  const window = windowTabs()
  return recordingKind(
    window.handle,
    () => held as unknown as TranscriptState,
    { runs: () => {} },
    fileOpeners({ fileKinds: async () => new Map() }),
    played,
  ).kind
}

describe('what a command asked over a recording tab is over', () => {
  it('is the file it plays, which is what a run is asked over', () => {
    const held = recording('talks/Ants.mp3', 4000, 9000)

    expect(kind(held).over!(held)).toStrictEqual({
      file: 'talks/Ants.mp3',
      source: 'recording',
    })
  })
})

describe('what a recording tab holds, as whoever answers for the person is told it', () => {
  it('is the file, how far it is written down, and how long it runs', () => {
    const held = recording('talks/Ants.mp3', 4000, 9000)

    expect(kind(held).attends!(held)).toStrictEqual({
      path: 'talks/Ants.mp3',
      recording: { transcribedDurationMs: 4000, durationMs: 9000 },
    })
  })
})

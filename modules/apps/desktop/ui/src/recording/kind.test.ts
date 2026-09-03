/**
 * What the window says about a recording tab, asked without a screen.
 *
 * How far a recording has been written down and how long it runs are both
 * milliseconds, and both are what is reported of the tab.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { recordingKind, type Held } from './kind'
import type { Transcript } from './transcript'
import { putting } from '../putting'
import { windowing } from '../windowing'

/** A recording open in a tab, as far as the window reads one. */
const recording = (path: string, heard: number, length: number) =>
  ({ path, heard: ref(heard), length: ref(length) }) as unknown as Held

/** The kind, made with a window that opens recordings this test hands it. */
const kind = (held: Held) => {
  const window = windowing()
  return recordingKind(
    window.host,
    () => held as unknown as Transcript,
    { runs: () => {} },
    putting({ standing: async () => new Map() }),
  ).kind
}

describe('what a command asked over a recording tab is over', () => {
  it('is the file it plays, which is what a run is asked over', () => {
    const held = recording('talks/Ants.mp3', 4000, 9000)

    expect(kind(held).at!(held)).toStrictEqual({
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
      at: 4000,
      of: 9000,
    })
  })
})

/**
 * The transcript as the editor lays it out: one line per cue, and what the tab
 * reports about whether there is any text at all.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useTranscriptLines } from './lines'
import type { Cue } from '../lib/cues'

const CUES: readonly Cue[] = [
  { text: 'A bell over the door.', from: 1_000, to: 3_000 },
  { text: 'Rain on the awning.', from: 3_000, to: 6_000 },
]

const lines = (cues: readonly Cue[], prose: string) => useTranscriptLines(ref(cues), ref(prose))

describe('a recording nothing was heard in and nothing was typed into', () => {
  it('has no lines, is not timed, and has nothing written', () => {
    const { spans, timed, hasText, times, transcribedDuration } = lines([], '')

    expect(spans.value).toStrictEqual([])
    expect(timed.value).toBe(false)
    expect(hasText.value).toBe(false)
    expect(times.value).toStrictEqual([])
    expect(transcribedDuration.value).toBe(0)
  })
})

describe('a page, which is prose with no times', () => {
  it('has a line for what is written, none of it timed, and is written', () => {
    const { spans, timed, hasText } = lines([], 'A page of prose.')

    expect(spans.value).toHaveLength(1)
    expect(spans.value[0]!.text).toBe('A page of prose.')
    expect(timed.value).toBe(false)
    expect(hasText.value).toBe(true)
  })
})

describe('a recording whose words were heard', () => {
  it('has a line to a cue, a moment for each, and is timed and written', () => {
    const { spans, timed, hasText, times, transcribedDuration } = lines(
      CUES,
      'A bell over the door.\nRain on the awning.',
    )

    expect(spans.value).toStrictEqual(CUES)
    expect(times.value).toHaveLength(2)
    expect(times.value[0]).not.toBe(times.value[1])
    expect(timed.value).toBe(true)
    expect(hasText.value).toBe(true)
    expect(transcribedDuration.value).toBe(6_000)
  })

  it('is still written, with a line left, when every word was taken from the screen', () => {
    const { spans, hasText } = lines(CUES, '')

    expect(spans.value).toHaveLength(1)
    expect(hasText.value).toBe(true)
  })
})

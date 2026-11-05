/**
 * A recording tab the window asks to light ranges of the transcript: playback goes to the cue of the
 * first range.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, reactive } from 'vue'
import RecordingTab from './RecordingTab.vue'
import { createPlayer, createRecordings, flush } from './model/fakes'
import type { TextRange } from './types'

vi.mock('@numen/ui', async (original) => ({
  ...(await original<Record<string, unknown>>()),
  NamedIcon: defineComponent({
    props: { name: { type: String, required: true } },
    setup:
      (props, { attrs }) =>
      () =>
        h('i', { ...attrs, 'data-icon': props.name }),
  }),
}))

const mountTab = (ranges: readonly TextRange[] | null) => {
  const setup = createRecordings()
  setup.state.cueTimes.set(22, 83_000)
  setup.state.cueTimes.set(140, 302_000)
  const { player, seeks } = createPlayer()
  const state = reactive({ path: 'talks/Ants.mp3', selector: null, ranges })
  const tab = mount(RecordingTab, {
    props: {
      path: 'talks/Ants.mp3',
      state,
      recordings: setup.recordings,
      how: { audioPlayer: player, plays: () => true },
    },
    attachTo: document.body,
  })
  return { tab, seeks, state }
}

describe('the ranges of a recording tab', () => {
  it('seeks to the cue of the first range the tab opens with', async () => {
    const { tab, seeks } = mountTab([
      { start: 22, end: 30 },
      { start: 140, end: 150 },
    ])
    await flush()

    expect(seeks.at(-1)?.[1]).toBe(83_000)
    tab.unmount()
  })

  it('seeks to the cue of the first range the window sets later', async () => {
    const { tab, seeks, state } = mountTab(null)
    await flush()

    state.ranges = [
      { start: 140, end: 150 },
      { start: 22, end: 30 },
    ]
    await flush()

    expect(seeks.at(-1)?.[1]).toBe(302_000)
    tab.unmount()
  })
})

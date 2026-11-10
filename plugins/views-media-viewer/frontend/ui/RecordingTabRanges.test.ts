/**
 * A recording tab whose transcript ranges the window highlights: playback seeks to the cue of the
 * first range.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, reactive } from 'vue'
import { formatErrorMessage, type Selector } from '@numen/plugin-sdk'
import RecordingTab from './RecordingTab.vue'
import { createPlayer, createRecordings, flush } from '../model/fakes'
import type { TextRange } from '../types'

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

  it('shows the failure of saving the place a range moved playback to', async () => {
    const setup = createRecordings()
    setup.state.cueTimes.set(22, 83_000)
    const { player } = createPlayer()
    const failure = new Error('the place store of talks/Ants.mp3 is full')
    let isFull = true
    const savePlace = vi.fn((selector: { t: [number, number] }) => {
      if (isFull && selector.t[0] === 83) {
        isFull = false
        throw failure
      }
    })
    const state = reactive({
      path: 'talks/Ants.mp3',
      selector: null,
      ranges: null as readonly TextRange[] | null,
      savePlace,
    })
    const tab = mount(RecordingTab, {
      props: {
        path: 'talks/Ants.mp3',
        state,
        recordings: setup.recordings,
        how: { audioPlayer: player, plays: () => true },
      },
      attachTo: document.body,
    })
    await flush()

    state.ranges = [{ start: 22, end: 30 }]
    await flush()

    expect(tab.find('[role="alert"]').text()).toBe(formatErrorMessage(failure))
    tab.unmount()
  })
})

describe('the place a recording tab saves when the window moves it', () => {
  const mountSavingTab = (ranges: readonly TextRange[] | null) => {
    const setup = createRecordings()
    setup.state.cueTimes.set(22, 83_000)
    setup.state.cueTimes.set(140, 66_000)
    const { player } = createPlayer()
    const savePlace = vi.fn()
    const state = reactive({
      path: 'talks/Ants.mp3',
      selector: null as Selector | null,
      ranges,
      savePlace,
    })
    const tab = mount(RecordingTab, {
      props: {
        path: 'talks/Ants.mp3',
        state,
        recordings: setup.recordings,
        how: { audioPlayer: player, plays: () => true },
      },
      attachTo: document.body,
    })
    return { tab, state, savePlace }
  }

  it('is the time of the first range the window sets, saved after the seek', async () => {
    const { tab, state, savePlace } = mountSavingTab(null)
    await flush()
    savePlace.mockClear()

    state.ranges = [
      { start: 140, end: 150 },
      { start: 22, end: 30 },
    ]
    await flush()

    expect(savePlace).toHaveBeenLastCalledWith({ t: [66, 66] })
    tab.unmount()
  })

  it('is the time of the first range the tab opens with, saved before anything plays', async () => {
    const { tab, savePlace } = mountSavingTab([{ start: 22, end: 30 }])
    await flush()

    expect(savePlace).toHaveBeenLastCalledWith({ t: [83, 83] })
    tab.unmount()
  })

  it('is the time a selector the window sets names, saved as soon as playback goes there', async () => {
    const { tab, state, savePlace } = mountSavingTab(null)
    await flush()
    savePlace.mockClear()

    state.selector = { t: [125, 125] }
    await flush()

    expect(savePlace).toHaveBeenLastCalledWith({ t: [125, 125] })
    tab.unmount()
  })
})

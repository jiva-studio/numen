/**
 * The place a recording tab saves when it opens at a restored place and when it closes, and the place
 * it reports to the window.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, reactive } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import RecordingTab from './RecordingTab.vue'
import { TALK_URL, createPlayer, createRecordings, flush } from '../model/fakes'

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

const OTHER_URL = 'http://127.0.0.1:1/token/v/talks/Bees.mp3'

const mountTab = (selector: Selector | null) => {
  const { recordings } = createRecordings()
  const { player } = createPlayer()
  const savePlace = vi.fn()
  const reportPlace = vi.fn()
  const state = reactive({ path: 'talks/Ants.mp3', selector, ranges: null, savePlace, reportPlace })
  const tab = mount(RecordingTab, {
    props: {
      path: 'talks/Ants.mp3',
      state,
      recordings,
      how: { audioPlayer: player, plays: () => true },
    },
    attachTo: document.body,
  })
  return { tab, player, savePlace, reportPlace }
}

describe('the place a recording tab opens at', () => {
  it('is not saved again, so the end of a restored place is kept', async () => {
    const { tab, savePlace } = mountTab({ t: [125, 130] })
    await flush()

    expect(savePlace).not.toHaveBeenCalled()
    tab.unmount()
  })
})

describe('the place a recording tab saves when it closes', () => {
  it('is the current time while it owns the shared player', async () => {
    const { tab, player, savePlace } = mountTab(null)
    await flush()
    player.seek(TALK_URL, 42_000)

    tab.unmount()

    expect(savePlace).toHaveBeenLastCalledWith({ t: [42, 42] })
  })

  it('is nothing while another recording owns the shared player', async () => {
    const { tab, player, savePlace } = mountTab({ t: [42, 42] })
    await flush()
    player.seek(OTHER_URL, 0)
    savePlace.mockClear()

    tab.unmount()

    expect(savePlace).not.toHaveBeenCalled()
  })
})

describe('the place a recording tab reports to the window', () => {
  it('is the current time in seconds when it closes while it owns the shared player', async () => {
    const { tab, player, reportPlace } = mountTab(null)
    await flush()
    player.seek(TALK_URL, 66_500)

    tab.unmount()

    expect(reportPlace).toHaveBeenLastCalledWith({ positionSeconds: 66.5 })
  })

  it('is a time past an hour, in seconds, when it closes there', async () => {
    const { tab, player, reportPlace } = mountTab(null)
    await flush()
    player.seek(TALK_URL, 3_723_900)

    tab.unmount()

    expect(reportPlace).toHaveBeenLastCalledWith({ positionSeconds: 3723.9 })
  })

  it('is nothing while another recording owns the shared player', async () => {
    const { tab, player, reportPlace } = mountTab({ t: [42, 42] })
    await flush()
    player.seek(OTHER_URL, 0)
    reportPlace.mockClear()

    tab.unmount()

    expect(reportPlace).not.toHaveBeenCalled()
  })
})

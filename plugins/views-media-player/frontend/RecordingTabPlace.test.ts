/**
 * The place a recording tab saves when it opens at a restored place and when it closes.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, reactive } from 'vue'
import RecordingTab from './RecordingTab.vue'
import { TALK_URL, createPlayer, createRecordings, flush } from './model/fakes'

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

const mountTab = (selector: Record<string, unknown> | null) => {
  const { recordings } = createRecordings()
  const { player } = createPlayer()
  const savePlace = vi.fn()
  const state = reactive({ path: 'talks/Ants.mp3', selector, ranges: null, savePlace })
  const tab = mount(RecordingTab, {
    props: {
      path: 'talks/Ants.mp3',
      state,
      recordings,
      how: { audioPlayer: player, plays: () => true },
    },
    attachTo: document.body,
  })
  return { tab, player, savePlace }
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
  it('is the time it is at while it holds the player', async () => {
    const { tab, player, savePlace } = mountTab(null)
    await flush()
    player.seek(TALK_URL, 42_000)

    tab.unmount()

    expect(savePlace).toHaveBeenLastCalledWith({ t: [42, 42] })
  })

  it('is nothing while another recording holds the player', async () => {
    const { tab, player, savePlace } = mountTab({ t: [42, 42] })
    await flush()
    player.seek(OTHER_URL, 0)
    savePlace.mockClear()

    tab.unmount()

    expect(savePlace).not.toHaveBeenCalled()
  })
})

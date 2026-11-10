/**
 * The place a recording tab reports to the window when it opens at a restored place or a search hit.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, reactive } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import RecordingTab from './RecordingTab.vue'
import { createPlayer, createRecordings, flush } from '../model/fakes'

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
  return { tab, savePlace, reportPlace }
}

describe('the place a recording tab reports when it opens', () => {
  it('is the start of the selector it opens at, before anything plays, and nothing is saved', async () => {
    const { tab, savePlace, reportPlace } = mountTab({ t: [125.5, 130] })
    await flush()

    expect(reportPlace).toHaveBeenCalledWith({ positionSeconds: 125.5 })
    expect(savePlace).not.toHaveBeenCalled()
    tab.unmount()
  })

  it('is nothing for a tab that opens with no selector', async () => {
    const { tab, reportPlace } = mountTab(null)
    await flush()

    expect(reportPlace).not.toHaveBeenCalled()
    tab.unmount()
  })
})

/**
 * A rendered recording tab: the player at the top, the transcript below it, and what can be queried
 * about the words.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, reactive } from 'vue'
import RecordingTab from './RecordingTab.vue'
import {
  CUES,
  DELETE_ENTRY,
  PROOFREAD_ENTRY,
  TALK_URL,
  TRANSCRIBE_ENTRY,
  createPlayer,
  createRecordings,
  flush,
} from './model/fakes'
import type { Recordings } from './types'
import { WORDS } from './words'

// The host loads the icon set lazily, so the test checks the name the tab passes to NamedIcon.
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

const mountTab = (
  recordings: Recordings,
  isPlayable = true,
  state = reactive({ path: 'talks/Ants.mp3' }),
) => {
  const { player, seeks } = createPlayer()
  const tab = mount(RecordingTab, {
    props: {
      path: 'talks/Ants.mp3',
      state,
      recordings,
      how: { audioPlayer: player, plays: () => isPlayable },
    },
    attachTo: document.body,
  })
  return { tab, seeks, state }
}

describe('a recording tab', () => {
  it('draws the player at the top and the words under it', async () => {
    const { recordings } = createRecordings()
    const { tab } = mountTab(recordings)
    await flush()

    expect(tab.find('.recording__player').exists()).toBe(true)
    expect(tab.find('button[aria-label="Play"]').exists()).toBe(true)
    expect(tab.find('.transcript__text').text()).toContain('The second thing said.')
    tab.unmount()
  })

  it('says it cannot play a recording of a kind the window does not play', async () => {
    const { recordings } = createRecordings()
    const { tab } = mountTab(recordings, false)
    await flush()

    expect(tab.find('.recording__player').exists()).toBe(false)
    expect(tab.find('.recording__note').text()).toBe(WORDS.unplayable)
    tab.unmount()
  })

  it('says nothing about playing while the recording is still being read', async () => {
    const { recordings } = createRecordings()
    let answer: (source: { mediaUrl: string; mediaType: string }) => void = () => undefined
    recordings.getSource = () => new Promise((resolve) => (answer = resolve))
    const { tab } = mountTab(recordings, false)
    await flush()

    expect(tab.find('.recording__note').exists()).toBe(false)
    expect(tab.text()).not.toContain(WORDS.unplayable)

    answer({ mediaUrl: TALK_URL, mediaType: 'audio/mpeg' })
    await flush()
    expect(tab.find('.recording__note').text()).toBe(WORDS.unplayable)
    tab.unmount()
  })

  it('offers to transcribe a recording nothing was heard in, and asks the plugin that reads it', async () => {
    const { recordings, state } = createRecordings([], [TRANSCRIBE_ENTRY])
    const { tab } = mountTab(recordings)
    await flush()

    const ask = tab.find('.transcript__ask')
    expect(ask.text()).toBe(TRANSCRIBE_ENTRY.title)
    await ask.trigger('click')
    await flush()

    expect(state.runs).toStrictEqual([TRANSCRIBE_ENTRY])
    tab.unmount()
  })

  it('offers no transcription where no plugin reads the recording', async () => {
    const { recordings } = createRecordings([], [])
    const { tab } = mountTab(recordings)
    await flush()

    expect(tab.find('.transcript__ask').exists()).toBe(false)
    expect(tab.find('.transcript__note').text()).toBe(WORDS.silence)
    tab.unmount()
  })

  it('deletes the transcript from the menu at the end of the strip', async () => {
    const { recordings, state } = createRecordings(CUES, [DELETE_ENTRY])
    const { tab } = mountTab(recordings)
    await flush()

    await tab.find('.media__more').trigger('click')
    const item = [...document.querySelectorAll('[role="menuitem"]')].find((one) =>
      one.textContent?.includes(DELETE_ENTRY.title),
    ) as HTMLElement | undefined
    expect(item).toBeDefined()
    item!.click()
    await flush()

    expect(state.runs).toStrictEqual([DELETE_ENTRY])
    tab.unmount()
  })

  it('lists every entry of the menu in the overflow menu when a transcript exists', async () => {
    const { recordings, state } = createRecordings(CUES, [
      TRANSCRIBE_ENTRY,
      PROOFREAD_ENTRY,
      DELETE_ENTRY,
    ])
    const { tab } = mountTab(recordings)
    await flush()

    await tab.find('.media__more').trigger('click')
    const titles = [...document.querySelectorAll('[role="menuitem"]')].map((one) =>
      one.textContent?.trim(),
    )
    expect(titles).toStrictEqual([
      TRANSCRIBE_ENTRY.title,
      PROOFREAD_ENTRY.title,
      DELETE_ENTRY.title,
    ])
    const icons = [...document.querySelectorAll('[role="menuitem"]')].map(
      (one) => one.querySelector('.media__menu-icon')?.getAttribute('data-icon') ?? '',
    )
    expect(icons).toStrictEqual([TRANSCRIBE_ENTRY.icon, PROOFREAD_ENTRY.icon, ''])
    const proofreadItem = [...document.querySelectorAll('[role="menuitem"]')].find(
      (one) => one.textContent?.trim() === PROOFREAD_ENTRY.title,
    ) as HTMLElement
    proofreadItem.click()
    await flush()
    expect(state.runs).toStrictEqual([PROOFREAD_ENTRY])

    await tab.find('.media__more').trigger('click')
    const retranscribeItem = [...document.querySelectorAll('[role="menuitem"]')].find(
      (one) => one.textContent?.trim() === TRANSCRIBE_ENTRY.title,
    ) as HTMLElement
    retranscribeItem.click()
    await flush()

    expect(state.runs).toStrictEqual([PROOFREAD_ENTRY, TRANSCRIBE_ENTRY])
    tab.unmount()
  })

  it('seeks to the cue of a search hit span', async () => {
    const setup = createRecordings()
    setup.state.cueTimes.set(22, 83_000)
    const { tab, seeks, state } = mountTab(setup.recordings)
    await flush()

    const focus = (
      state as { focusSpans?: (...spans: { start: number; end: number }[]) => Promise<void> }
    ).focusSpans
    await focus?.({ start: 22, end: 30 })

    expect(seeks.at(-1)?.[1]).toBe(83_000)
    tab.unmount()
  })
})

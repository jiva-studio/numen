/**
 * A recording tab drawn: the player at the top, the transcript under it, and what can be asked
 * about the words.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { reactive } from 'vue'
import RecordingTab from './RecordingTab.vue'
import {
  CUES,
  DELETE_ENTRY,
  TALK_URL,
  TRANSCRIBE_ENTRY,
  createPlayer,
  createRecordings,
  flush,
} from './model/fakes'
import type { Recordings } from './types'
import { WORDS } from './words'

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
    let answer: (source: { mediaUrl: string; mediaType: string }) => void = () => {}
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
    expect(ask.text()).toBe(WORDS.transcribe)
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
      one.textContent?.includes(WORDS.deleteText),
    ) as HTMLElement | undefined
    expect(item).toBeDefined()
    item!.click()
    await flush()

    expect(state.runs).toStrictEqual([DELETE_ENTRY])
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

/**
 * A recording tab drawn.
 *
 * The words stand in an editor, and the moment each was said stands in its
 * gutter. The transcript arrives after the tab is drawn, so the times have to
 * reach an editor that was not there when they were first shown.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import RecordingTab from './RecordingTab.vue'
import { asking, listening, type Cue, type Recordings } from './listening'
import type { Player } from './playing'
import { WORDS } from './words'

const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 83_000, to: 85_000 },
]

/** A recording that answers only once the tab has been drawn. */
function talk(cues: readonly Cue[] = CUES): Recordings {
  return {
    listened: async () => ({
      length: 85_000,
      heard: 85_000,
      media: 'http://127.0.0.1:1/w/v/talk.mp3',
      type: 'audio/mpeg',
    }),
    cues: async () => ({ cues, editable: true }),
    writes: async () => {},
    plays: async () => null,
  }
}

/** The one player the window has, faked: nothing here makes a sound. */
function played(): Player {
  const address = ref('')
  return {
    address,
    at: ref(0),
    length: ref(0),
    playing: ref(false),
    failed: ref(''),
    load: (wanted) => void (address.value = wanted),
    play: (wanted) => void (address.value = wanted),
    pause: () => {},
    seek: (wanted) => void (address.value = wanted),
  }
}

// The window plays what it is given, so the controls stand where they stand
// and the only note drawn is the one about the words.
beforeEach(() => asking(() => true))

/** Everything asked for has been answered and everything drawn has settled. */
const settled = async () => {
  await new Promise((done) => setTimeout(done, 0))
  await new Promise((done) => setTimeout(done, 0))
}

describe('a recording tab', () => {
  it('stands the moment each line was said in the editor gutter', async () => {
    const held = listening(talk(), 'talks/Ants.mp3', played())
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const times = drawn.findAll('.cm-times .cm-gutterElement').map((one) => one.text())
    expect(times).toContain('0:00')
    expect(times).toContain('1:23')

    drawn.unmount()
  })
})

describe('a recording nothing was heard in', () => {
  it('says so, and draws no editor at all', async () => {
    const held = listening(talk([]), 'talks/Ants.mp3', played())
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.recording__transcript').exists()).toBe(false)
    expect(drawn.find('.recording__note').text()).toBe(WORDS.silence)

    drawn.unmount()
  })

  it('says a run is going while one is, and still draws no editor', async () => {
    const held = listening(talk([]), 'talks/Ants.mp3', played())
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()

    held.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.recording__transcript').exists()).toBe(false)
    expect(drawn.find('.recording__note').text()).toBe(WORDS.transcribing)

    drawn.unmount()
  })
})

describe('a transcript still growing', () => {
  it('draws the words and says nothing under them', async () => {
    const held = listening(talk(), 'talks/Ants.mp3', played())
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    held.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.recording__transcript').exists()).toBe(true)
    expect(drawn.find('.recording__note').exists()).toBe(false)

    drawn.unmount()
  })
})

describe('a recording tab drawn again', () => {
  it('still stands the times in its gutter', async () => {
    const held = listening(talk(), 'talks/Ants.mp3', played())
    const first = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()
    await first.vm.$nextTick()
    await settled()
    expect(first.findAll('.cm-times .cm-gutterElement').length).toBeGreaterThan(0)

    // A tab moved between panes is unmounted and drawn again, holding the
    // same recording. Nothing about the words changes as it moves.
    first.unmount()
    const again = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()
    await again.vm.$nextTick()
    await settled()

    const times = again.findAll('.cm-times .cm-gutterElement').map((one) => one.text())
    expect(times).toContain('0:00')
    expect(times).toContain('1:23')

    again.unmount()
  })
})

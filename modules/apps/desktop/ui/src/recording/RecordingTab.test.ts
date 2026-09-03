/**
 * A recording tab drawn.
 *
 * The words stand in an editor, and the moment each was said stands in its
 * gutter. The transcript arrives after the tab is drawn, so the times have to
 * reach an editor that was not there when they were first shown.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import RecordingTab from './RecordingTab.vue'
import { transcribed } from './kind'
import { asking, listening, type Cue, type Recordings } from './listening'
import type { Player } from './playing'
import { cannotRun, runsAgain } from '../commanding'
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

/** One recording tab, and every run it asked the window for. */
function tab(cues: readonly Cue[] = CUES) {
  const asked: string[] = []
  const held = transcribed(listening(talk(cues), 'talks/Ants.mp3', played()), {
    runs: (id, path, called) => void asked.push(`${id} ${path} ${called}`),
  })
  return { held, asked }
}

// The window plays what it is given, so the controls stand where they stand
// and the only note drawn is the one about the words.
beforeEach(() => asking(() => true))

// A build that answered it cannot do a run offers it nowhere after that, and
// the answer outlives the tab that got it.
afterEach(() => runsAgain())

/** Everything asked for has been answered and everything drawn has settled. */
const settled = async () => {
  await new Promise((done) => setTimeout(done, 0))
  await new Promise((done) => setTimeout(done, 0))
}

describe('a recording tab', () => {
  it('stands the moment each line was said in the editor gutter', async () => {
    const { held } = tab()
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

describe('a recording with no transcript', () => {
  // The button says there is no transcript, so nothing says it twice.
  it('draws no editor at all, and offers the run in place of the words', async () => {
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.recording__transcript').exists()).toBe(false)
    expect(drawn.find('.recording__ask').text()).toBe(WORDS.transcribe)
    expect(drawn.find('.recording__note').exists()).toBe(false)

    drawn.unmount()
  })

  // Where the run cannot be asked for, what there is to say is said.
  it('says there is no transcript where the run cannot be asked for', async () => {
    cannotRun('transcribe')
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.recording__ask').exists()).toBe(false)
    expect(drawn.find('.recording__note').text()).toBe(WORDS.silence)

    drawn.unmount()
  })

  it('offers the run under the player', async () => {
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.recording__ask').text()).toBe(WORDS.transcribe)

    drawn.unmount()
  })

  it('asks the window for that run when it is pressed', async () => {
    const { held, asked } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    await drawn.find('.recording__ask').trigger('click')

    expect(asked).toStrictEqual(['transcribe talks/Ants.mp3 Ants.mp3'])

    drawn.unmount()
  })

  it('offers nothing where this build cannot do the run at all', async () => {
    cannotRun('transcribe')
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.recording__ask').exists()).toBe(false)
    expect(drawn.find('.recording__note').text()).toBe(WORDS.silence)

    drawn.unmount()
  })

  it('says a run is going while one is, and offers none beside it', async () => {
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()

    held.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.recording__transcript').exists()).toBe(false)
    expect(drawn.find('.recording__note').text()).toBe(WORDS.transcribing)
    expect(drawn.find('.recording__ask').exists()).toBe(false)

    drawn.unmount()
  })

  // The player heads the pane, and a run going changes only what is below it.
  it('draws the player the same while a run goes as before it began', async () => {
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const head = drawn.find('.recording__head').html()

    held.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.recording__head').html()).toBe(head)
    expect(drawn.find('.recording__below').text()).toBe(WORDS.transcribing)

    drawn.unmount()
  })

  // The player is what a recording is for, and a build with nothing to play it
  // with says so where the controls would stand.
  it('says a recording it cannot play at all is one', async () => {
    asking(() => false)
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.text()).toContain(WORDS.unplayable)
    expect(drawn.find('.recording__ask').text()).toBe(WORDS.transcribe)

    drawn.unmount()
  })
})

describe('a transcript still growing', () => {
  it('draws the words and says nothing under them', async () => {
    const { held } = tab()
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

describe('the menu at the end of the player strip', () => {
  /** What the menu offers, as it is drawn. */
  const offered = () =>
    [...document.body.querySelectorAll('.menu__item')].map((one) => one.textContent?.trim() ?? '')

  // The two controls over the words stand together, at their own spacing.
  it('stands beside the follow control, in one group at the end of the strip', async () => {
    const { held } = tab()
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const deeds = drawn.get('.recording__deeds')
    expect(deeds.find('.recording__follow').exists()).toBe(true)
    expect(deeds.find('.recording__more').exists()).toBe(true)

    drawn.unmount()
  })

  // Putting the words right stands above taking them away, so the one that
  // cannot be undone is last.
  it('offers the words put right and taken away where they already stand', async () => {
    const { held } = tab()
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const more = drawn.get('.recording__more')
    expect(more.attributes('aria-haspopup')).toBe('menu')
    expect(offered()).toStrictEqual([])

    await more.trigger('click')

    expect(offered()).toStrictEqual([WORDS.proofread, WORDS.drop])

    drawn.unmount()
  })

  it('asks the window for the run behind whichever item is chosen', async () => {
    const { held, asked } = tab()
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const chooses = async (text: string) => {
      await drawn.get('.recording__more').trigger('click')
      const chosen = [...document.body.querySelectorAll<HTMLElement>('.menu__item')].find(
        (one) => one.textContent?.trim() === text,
      )
      chosen?.click()
      await drawn.vm.$nextTick()
    }

    await chooses(WORDS.proofread)
    await chooses(WORDS.drop)

    expect(asked).toStrictEqual([
      'proofread talks/Ants.mp3 Ants.mp3',
      'dropTranscript talks/Ants.mp3 Ants.mp3',
    ])
    expect(offered()).toStrictEqual([])

    drawn.unmount()
  })

  // A run appends to the words, and what is being appended to is not taken
  // away underneath it.
  it('is not drawn while a run is writing the words down', async () => {
    const { held } = tab()
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    held.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.recording__more').exists()).toBe(false)

    drawn.unmount()
  })

  // Each item stands only where this build can do the run behind it, and the
  // menu itself only where an item stands.
  it('drops an item this build cannot do at all, and goes where none is left', async () => {
    cannotRun('dropTranscript')
    const { held } = tab()
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    await drawn.get('.recording__more').trigger('click')
    expect(offered()).toStrictEqual([WORDS.proofread])

    cannotRun('proofread')
    await drawn.vm.$nextTick()

    expect(drawn.find('.recording__more').exists()).toBe(false)

    drawn.unmount()
  })

  // Nothing can be asked over words that are not there.
  it('is not drawn where the recording has no transcript', async () => {
    const { held } = tab([])
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.recording__more').exists()).toBe(false)

    drawn.unmount()
  })
})

describe('a recording with a transcript', () => {
  it('draws nothing above the words but the player strip', async () => {
    const { held } = tab()
    const drawn = mount(RecordingTab, { props: { held }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.recording__ask').exists()).toBe(false)

    drawn.unmount()
  })
})

describe('a recording tab drawn again', () => {
  it('still stands the times in its gutter', async () => {
    const { held } = tab()
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

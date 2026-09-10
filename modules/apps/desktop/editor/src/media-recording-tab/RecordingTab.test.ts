/**
 * A recording tab drawn.
 *
 * The words stand in an editor, and the moment each was said stands in its
 * gutter. The transcript arrives after the tab is drawn, so the times have to
 * reach an editor that was not there when they were first shown.
 */
// The editor measures the text it drew on a frame of its own, after the test
// that mounted it is over.
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { runSupport } from '../shared/command/runs'
import RecordingTab from './RecordingTab.vue'
import { transcribed } from '../shared/media/kind'
import { transcript, type Recordings } from '../shared/media/transcript'
import type { Cue } from '../shared/media/cues'
import type { MediaTypeProbe, Player } from '../shared/media/player'
import { WORDS } from '../shared/media/words'

const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 83_000, to: 85_000 },
]

/** A recording that answers only once the tab has been drawn. */
function talk(cues: readonly Cue[] = CUES): Recordings {
  return {
    getSummary: async () => ({
      duration: 85_000,
      mediaUrl: 'http://127.0.0.1:1/w/v/talk.mp3',
      mediaType: 'audio/mpeg',
      url: '',
    }),
    getTaskStates: async () => ({ transcript: 'done' }),
    readTranscript: async () => ({ cues, editable: true, prose: '' }),
    readArticle: async () => ({ cues: [], editable: true, prose: '' }),
    writeTranscript: async () => {},
    findCueTime: async () => null,
  }
}

/** The one player the window has, faked: nothing here makes a sound. */
function played(): Player {
  const address = ref('')
  return {
    address,
    at: ref(0),
    duration: ref(0),
    playing: ref(false),
    failed: ref(''),
    load: (wanted) => void (address.value = wanted),
    play: (wanted) => void (address.value = wanted),
    pause: () => {},
    seek: (wanted) => void (address.value = wanted),
  }
}

/**
 * One recording tab, and every run it asked the window for. The window plays
 * what it is given, so the controls stand where they stand and the only note
 * drawn is the one about the words.
 */
function tab(
  cues: readonly Cue[] = CUES,
  plays: MediaTypeProbe = () => true,
  canRun: (run: string) => boolean = () => true,
) {
  const asked: string[] = []
  const state = transcribed(
    transcript(talk(cues), 'talks/Ants.mp3', { through: played(), plays }),
    { runs: (id, path, called) => void asked.push(`${id} ${path} ${called}`), canRun },
  )
  return { state, asked }
}

/** A window this build has told it can do no run at all. */
const nothing = () => false

/** Everything asked for has been answered and everything drawn has settled. */
const settled = async () => {
  await new Promise((done) => setTimeout(done, 0))
  await new Promise((done) => setTimeout(done, 0))
}

describe('a recording tab', () => {
  it('stands the moment each line was said in the editor gutter', async () => {
    const { state } = tab()
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

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
    const { state } = tab([])
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.transcript__text').exists()).toBe(false)
    expect(drawn.find('.transcript__ask').text()).toBe(WORDS.transcribe)
    expect(drawn.find('.transcript__note').exists()).toBe(false)

    drawn.unmount()
  })

  // Where the run cannot be asked for, what there is to say is said.
  it('says there is no transcript where the run cannot be asked for', async () => {
    const { state } = tab([], undefined, nothing)
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.transcript__ask').exists()).toBe(false)
    expect(drawn.find('.transcript__note').text()).toBe(WORDS.silence)

    drawn.unmount()
  })

  it('offers the run under the player', async () => {
    const { state } = tab([])
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.transcript__ask').text()).toBe(WORDS.transcribe)

    drawn.unmount()
  })

  it('asks the window for that run when it is pressed', async () => {
    const { state, asked } = tab([])
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    await drawn.find('.transcript__ask').trigger('click')

    expect(asked).toStrictEqual(['transcribe talks/Ants.mp3 Ants.mp3'])

    drawn.unmount()
  })

  it('offers nothing where this build cannot do the run at all', async () => {
    const { state } = tab([], undefined, nothing)
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.transcript__ask').exists()).toBe(false)
    expect(drawn.find('.transcript__note').text()).toBe(WORDS.silence)

    drawn.unmount()
  })

  it('says a run is going while one is, and offers none beside it', async () => {
    const { state } = tab([])
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()

    state.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.transcript__text').exists()).toBe(false)
    expect(drawn.find('.transcript__note').text()).toBe(WORDS.transcribing)
    expect(drawn.find('.transcript__ask').exists()).toBe(false)

    drawn.unmount()
  })

  // The player heads the pane, and a run going changes only what is below it.
  it('draws the player the same while a run goes as before it began', async () => {
    const { state } = tab([])
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const head = drawn.find('.media__head').html()

    state.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.media__head').html()).toBe(head)
    expect(drawn.find('.transcript').text()).toBe(WORDS.transcribing)

    drawn.unmount()
  })

  // The player is what a recording is for, and a build with nothing to play it
  // with says so where the controls would stand.
  it('says a recording it cannot play at all is one', async () => {
    const { state } = tab([], () => false)
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.text()).toContain(WORDS.unplayable)
    expect(drawn.find('.transcript__ask').text()).toBe(WORDS.transcribe)

    drawn.unmount()
  })
})

describe('a transcript still growing', () => {
  it('draws the words and says nothing under them', async () => {
    const { state } = tab()
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    state.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.transcript__text').exists()).toBe(true)
    expect(drawn.find('.transcript__note').exists()).toBe(false)

    drawn.unmount()
  })
})

describe('the menu at the end of the player strip', () => {
  /** What the menu offers, as it is drawn. */
  const offered = () =>
    [...document.body.querySelectorAll('.menu__item')].map((one) => one.textContent?.trim() ?? '')

  // The two controls over the words stand together, at their own spacing.
  it('stands beside the follow control, in one group at the end of the strip', async () => {
    const { state } = tab()
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const actions = drawn.get('.media__actions')
    expect(actions.find('.media__follow').exists()).toBe(true)
    expect(actions.find('.media__more').exists()).toBe(true)

    drawn.unmount()
  })

  // Putting the words right stands above taking them away, so the one that
  // cannot be undone is last.
  it('offers the words put right and taken away where they already stand', async () => {
    const { state } = tab()
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const more = drawn.get('.media__more')
    expect(more.attributes('aria-haspopup')).toBe('menu')
    expect(offered()).toStrictEqual([])

    await more.trigger('click')

    expect(offered()).toStrictEqual([WORDS.proofread, WORDS.deleteText])

    drawn.unmount()
  })

  it('asks the window for the run behind whichever item is chosen', async () => {
    const { state, asked } = tab()
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    const chooses = async (text: string) => {
      await drawn.get('.media__more').trigger('click')
      const chosen = [...document.body.querySelectorAll<HTMLElement>('.menu__item')].find(
        (one) => one.textContent?.trim() === text,
      )
      chosen?.click()
      await drawn.vm.$nextTick()
    }

    await chooses(WORDS.proofread)
    await chooses(WORDS.deleteText)

    expect(asked).toStrictEqual([
      'proofread talks/Ants.mp3 Ants.mp3',
      'deleteText talks/Ants.mp3 Ants.mp3',
    ])
    expect(offered()).toStrictEqual([])

    drawn.unmount()
  })

  // A run appends to the words, and what is being appended to is not taken
  // away underneath it.
  it('is not drawn while a run is writing the words down', async () => {
    const { state } = tab()
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()
    await drawn.vm.$nextTick()
    await settled()

    state.ticks(true)
    await settled()
    await drawn.vm.$nextTick()

    expect(drawn.find('.media__more').exists()).toBe(false)

    drawn.unmount()
  })

  // Each item stands only where this build can do the run behind it, and the
  // menu itself only where an item stands.
  it('drops an item this build cannot do at all, and goes where none is left', async () => {
    const runs = runSupport()
    runs.cannotRun('deleteText')
    const { state } = tab(CUES, undefined, (run) => runs.canRun(run))
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    await drawn.get('.media__more').trigger('click')
    expect(offered()).toStrictEqual([WORDS.proofread])

    runs.cannotRun('proofread')
    await drawn.vm.$nextTick()

    expect(drawn.find('.media__more').exists()).toBe(false)

    drawn.unmount()
  })

  // Nothing can be asked over words that are not there.
  it('is not drawn where the recording has no transcript', async () => {
    const { state } = tab([])
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.media__more').exists()).toBe(false)

    drawn.unmount()
  })
})

describe('a recording with a transcript', () => {
  it('draws nothing above the words but the player strip', async () => {
    const { state } = tab()
    const drawn = mount(RecordingTab, { props: { state }, attachTo: document.body })

    await settled()
    await drawn.vm.$nextTick()
    await settled()

    expect(drawn.find('.transcript__ask').exists()).toBe(false)

    drawn.unmount()
  })
})

describe('a recording tab drawn again', () => {
  it('still stands the times in its gutter', async () => {
    const { state } = tab()
    const first = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()
    await first.vm.$nextTick()
    await settled()
    expect(first.findAll('.cm-times .cm-gutterElement').length).toBeGreaterThan(0)

    // A tab moved between panes is unmounted and drawn again, holding the
    // same recording. Nothing about the words changes as it moves.
    first.unmount()
    const again = mount(RecordingTab, { props: { state }, attachTo: document.body })
    await settled()
    await again.vm.$nextTick()
    await settled()

    const times = again.findAll('.cm-times .cm-gutterElement').map((one) => one.text())
    expect(times).toContain('0:00')
    expect(times).toContain('1:23')

    again.unmount()
  })
})

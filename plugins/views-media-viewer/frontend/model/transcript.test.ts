/**
 * One recording as its tab reads it.
 */
import { describe, expect, it } from 'vitest'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { useTranscript } from './transcript'
import { CUES, TALK_URL, createPlayer, createRecordings, flush } from './fakes'
import { WORDS } from '../words'

const open = (setup = createRecordings()) => {
  const { player, seeks } = createPlayer()
  const tab = useTranscript(setup.recordings, 'talks/Ants.mp3', {
    audioPlayer: player,
    plays: () => true,
    idleMs: 1,
  })
  return { tab, player, seeks, ...setup }
}

describe('a recording firstRead', () => {
  it('is played from the address the application serves it at, and loaded into an empty player', async () => {
    const { tab, player } = open()
    await tab.content.firstRead
    expect(tab.playback.source.url.value).toBe(TALK_URL)
    expect(player.state.url.value).toBe(TALK_URL)
  })

  it('says it is loading until the answer lands, and no longer', async () => {
    const { tab } = open()
    expect(tab.content.isLoading.value).toBe(true)
    await tab.content.firstRead
    expect(tab.content.isLoading.value).toBe(false)
  })

  it('shows its words one cue to a line', async () => {
    const { tab } = open()
    await tab.content.firstRead
    expect(tab.draft.prose.value).toBe('The first thing said.\nThe second thing said.')
    expect(tab.lines.times.value).toStrictEqual(['0:00', '1:23'])
  })

  it('says what went wrong where the transcript could not be read', async () => {
    const setup = createRecordings()
    setup.state.readError = new Error('the plugin is not there')
    const { tab } = open(setup)
    await tab.content.firstRead
    expect(tab.content.error.value).toBe(formatErrorMessage(new Error('the plugin is not there')))
    expect(tab.content.isLoading.value).toBe(false)
  })
})

describe('a live text without times', () => {
  it('shows as prose without times, is not edited, and offers no Transcribe', async () => {
    const setup = createRecordings([], true)
    setup.state.article = 'A tidal mill stored seawater.\nIt turned at the ebb.'
    const { tab } = open(setup)
    await tab.content.firstRead

    expect(tab.draft.prose.value).toBe('A tidal mill stored seawater.\nIt turned at the ebb.')
    expect(tab.lines.timed.value).toBe(false)
    expect(tab.content.isEditable.value).toBe(false)
    expect(tab.options.isStartable.value).toBe(false)
  })
})

describe('creating the text of a recording', () => {
  it('is offered where nothing was heard and a plugin creates the text of the file', async () => {
    const { tab } = open(createRecordings([], true))
    await tab.content.firstRead
    expect(tab.options.isStartable.value).toBe(true)
    expect(tab.content.note.value).toBe(WORDS.silence)
  })

  it('is not offered where no plugin creates the text of the file', async () => {
    const { tab } = open(createRecordings([], false))
    await tab.content.firstRead
    expect(tab.options.isStartable.value).toBe(false)
  })

  it('is not offered where there are words', async () => {
    const { tab } = open(createRecordings(CUES, true))
    await tab.content.firstRead
    expect(tab.options.isStartable.value).toBe(false)
  })

  it('runs the command that creates the text, and reads the words again after it', async () => {
    const setup = createRecordings([], true)
    const { tab, state } = open(setup)
    await tab.content.firstRead
    state.cues = CUES

    await tab.options.start()

    expect(state.textRuns).toBe(1)
    expect(tab.content.cues.value).toStrictEqual(CUES)
  })

  it('shows the error of a run that failed in the tab', async () => {
    const { tab, state } = open(createRecordings([], true))
    await tab.content.firstRead
    state.textError = new Error('the recording has no audio')

    await tab.options.start()

    expect(tab.content.error.value).toBe(formatErrorMessage(state.textError))
  })
})

describe('a transcript still growing', () => {
  it('is read again while work on it goes, once more when it ends, and is not edited meanwhile', async () => {
    const { tab, state } = open(createRecordings([], true))
    await tab.content.firstRead
    const before = state.reads

    state.cues = [CUES[0]!]
    state.onWork?.(true)
    await flush()
    expect(tab.content.note.value).toBe('')
    expect(tab.content.isEditable.value).toBe(false)
    state.cues = CUES
    state.onWork?.(false)
    await flush()

    expect(state.reads).toBe(before + 2)
    expect(tab.content.cues.value).toStrictEqual(CUES)
    expect(tab.content.isEditable.value).toBe(true)
  })

  it('says a run is going where there are no words yet', async () => {
    const { tab, state } = open(createRecordings([], true))
    await tab.content.firstRead
    state.onWork?.(true)
    await flush()
    expect(tab.content.note.value).toBe(WORDS.transcribing)
  })

  it('is left alone where work ended that never began', async () => {
    const { tab, state } = open()
    await tab.content.firstRead
    const before = state.reads
    state.onWork?.(false)
    await flush()
    expect(state.reads).toBe(before)
  })
})

describe('the words as a person edits them', () => {
  it('are written as cues once the typing has been still', async () => {
    const { tab, state } = open()
    await tab.content.firstRead
    tab.draft.setProse('The first thing said.\nThe second thing, put right.')
    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(state.writes).toStrictEqual([
      [CUES[0], { text: 'The second thing, put right.', from: 83_000, to: 85_000 }],
    ])
  })
})

describe('a recording firstRead at a place in its words', () => {
  it('seeks to the cue of the first span', async () => {
    const setup = createRecordings()
    setup.state.cueTimes.set(22, 83_000)
    const { tab, seeks } = open(setup)
    await tab.navigation.seekToSpans({ start: 22, end: 30 })
    expect(seeks.at(-1)).toStrictEqual([TALK_URL, 83_000])
  })

  it('keeps the position when no cue holds the span', async () => {
    const { tab, seeks } = open()
    await tab.navigation.seekToSpans({ start: 900, end: 910 })
    expect(seeks).toStrictEqual([])
  })

  it('seeks to the latest request when the earlier one resolves last', async () => {
    const setup = createRecordings()
    const resolvers = new Map<number, (ms: number) => void>()
    setup.recordings.findCueTime = (_, span) =>
      new Promise((resolve) => resolvers.set(span.start, resolve))
    const { tab, seeks } = open(setup)
    await tab.content.firstRead

    const earlier = tab.navigation.seekToSpans({ start: 22, end: 30 })
    const later = tab.navigation.seekToSpans({ start: 140, end: 150 })
    await flush()
    resolvers.get(140)?.(302_000)
    expect(await later).toBe(302_000)
    resolvers.get(22)?.(83_000)

    expect(await earlier).toBeNull()
    expect(seeks.at(-1)).toStrictEqual([TALK_URL, 302_000])
  })
})

describe('a recording whose file the vault changes', () => {
  it('reads its words again, and shows none once the transcript is deleted', async () => {
    const { tab, state } = open()
    await tab.content.firstRead
    state.cues = []

    state.onChange?.()
    await flush()

    expect(tab.content.cues.value).toStrictEqual([])
    expect(tab.lines.hasText.value).toBe(false)
  })
})

describe('a recording tab that closes', () => {
  it('stops listening for work and holds no words', async () => {
    const { tab, state } = open()
    await tab.content.firstRead
    tab.lifecycle.close()
    expect(state.isWatched).toBe(false)
    expect(tab.content.cues.value).toStrictEqual([])
  })
})

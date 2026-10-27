/**
 * One recording as its tab reads it.
 */
import { describe, expect, it } from 'vitest'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { findOffers, useTranscript } from './transcript'
import {
  CUES,
  DELETE_ENTRY,
  TALK_URL,
  TRANSCRIBE_ENTRY,
  createPlayer,
  createRecordings,
  flush,
} from './fakes'
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
    expect(tab.playback.url.value).toBe(TALK_URL)
    expect(player.url.value).toBe(TALK_URL)
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

describe('what the menu offers', () => {
  it('takes text made from the file as transcribing and from the live version as proofreading', () => {
    const proofread = {
      ...TRANSCRIBE_ENTRY,
      pluginId: 'md.test.proofreading',
      versionId: DELETE_ENTRY.versionId,
      isLive: true,
    }
    const offers = findOffers([TRANSCRIBE_ENTRY, proofread, DELETE_ENTRY])
    expect(offers.transcribe).toBe(TRANSCRIBE_ENTRY)
    expect(offers.proofread).toBe(proofread)
    expect(offers.deleteText).toBe(DELETE_ENTRY)
  })

  it('is transcribing where nothing was heard, and not deleting', async () => {
    const { tab } = open(createRecordings([], [TRANSCRIBE_ENTRY]))
    await tab.content.firstRead
    expect(tab.offers.isTranscribable.value).toBe(true)
    expect(tab.offers.isDeletable.value).toBe(false)
    expect(tab.content.note.value).toBe(WORDS.silence)
  })

  it('is deleting the live version where there are words, and not transcribing', async () => {
    const { tab } = open(createRecordings(CUES, [TRANSCRIBE_ENTRY, DELETE_ENTRY]))
    await tab.content.firstRead
    expect(tab.offers.isTranscribable.value).toBe(false)
    expect(tab.offers.isDeletable.value).toBe(true)
  })

  it('offers no deletion of a version that is not the live one', async () => {
    const { tab } = open(createRecordings(CUES, [{ ...DELETE_ENTRY, isLive: false }]))
    await tab.content.firstRead
    expect(tab.offers.isDeletable.value).toBe(false)
  })

  it('carries the entry out, and reads the words again after a deletion', async () => {
    const setup = createRecordings(CUES, [DELETE_ENTRY])
    const { tab, state } = open(setup)
    await tab.content.firstRead
    state.cues = []

    await tab.offers.deleteTranscript()

    expect(state.runs).toStrictEqual([DELETE_ENTRY])
    expect(tab.content.cues.value).toStrictEqual([])
  })
})

describe('a transcript still growing', () => {
  it('is read again while work on it goes, once more when it ends, and is not edited meanwhile', async () => {
    const { tab, state } = open(createRecordings([], [TRANSCRIBE_ENTRY]))
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
    const { tab, state } = open(createRecordings([], [TRANSCRIBE_ENTRY]))
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
    await tab.navigation.reach({ start: 22, end: 30 })
    expect(seeks.at(-1)).toStrictEqual([TALK_URL, 83_000])
  })

  it('keeps the position when no cue holds the span', async () => {
    const { tab, seeks } = open()
    await tab.navigation.reach({ start: 900, end: 910 })
    expect(seeks).toStrictEqual([])
  })
})

describe('a recording tab that closes', () => {
  it('stops listening for work and holds no words', async () => {
    const { tab, state } = open()
    await tab.content.firstRead
    tab.close()
    expect(state.isWatched).toBe(false)
    expect(tab.content.cues.value).toStrictEqual([])
  })
})

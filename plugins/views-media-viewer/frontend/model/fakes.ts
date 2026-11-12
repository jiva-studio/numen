/** A recording and its transcript, held in memory for the tests of a recording tab. */
import { ref } from 'vue'
import type { Recordings, Cue } from '../types'
import type { Player } from './player'

export const TALK_URL = 'http://127.0.0.1:1/token/v/talks/Ants.mp3'

export const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 83_000, to: 85_000 },
]

/**
 * A recording that tests modify as the application would. `setWorking`
 * reports that work on it began or ended, as the window's tasks do.
 */
export function createRecordings(cues: readonly Cue[] = CUES, canCreateText = false) {
  const state = {
    cues,
    canCreateText,
    /** The prose of a live text without times, and empty where the live text has times. */
    article: '',
    reads: 0,
    writes: [] as (readonly Cue[])[],
    textRuns: 0,
    cueTimes: new Map<number, number>(),
    readError: null as Error | null,
    textError: null as Error | null,
    onWork: null as ((isWorking: boolean) => void) | null,
    onChange: null as (() => void) | null,
    isWatched: false,
  }
  const recordings: Recordings = {
    getSource: async () => ({ mediaUrl: TALK_URL, mediaType: 'audio/mpeg', pageUrl: '' }),
    readTranscript: async () => {
      state.reads++
      if (state.readError) throw state.readError
      return state.cues
    },
    readArticle: async () => state.article,
    writeTranscript: async (_, next) => {
      state.writes.push(next)
    },
    findCueTime: async (_, span) => state.cueTimes.get(span.start) ?? null,
    canCreateText: async () => state.canCreateText,
    createText: async () => {
      state.textRuns++
      if (state.textError) throw state.textError
    },
    watchChanges: (_, onChange) => {
      state.onChange = onChange
      return () => {
        state.onChange = null
      }
    },
    watchWork: (_, onChange) => {
      state.onWork = onChange
      state.isWatched = true
      return () => {
        state.isWatched = false
      }
    },
  }
  return { recordings, state }
}

/** The one player the window has, faked: nothing here makes a sound. */
export function createPlayer() {
  const url = ref('')
  const position = ref(0)
  const seeks: [string, number][] = []
  /** The containers the element was drawn in, in order, and released from. */
  const surfaces: HTMLElement[] = []
  const released: HTMLElement[] = []
  const player: Player = {
    state: { url, position, duration: ref(0), isPlaying: ref(false), error: ref('') },
    load: (next) => {
      url.value = next
    },
    play: (next) => {
      url.value = next
    },
    pause: () => undefined,
    seek: (next, ms) => {
      url.value = next
      position.value = ms
      seeks.push([next, ms])
    },
    getDuration: async () => 0,
    surface: {
      attach: (container) => {
        surfaces.push(container)
      },
      release: (container) => {
        released.push(container)
      },
    },
  }
  return { player, seeks, surfaces, released }
}

/** Resolves once every pending request has been answered. */
export const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

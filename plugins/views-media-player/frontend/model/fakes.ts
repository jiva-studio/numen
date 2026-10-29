/** A recording, its transcript and its menu, held in memory for the tests of a recording tab. */
import { ref } from 'vue'
import type { Cue } from '../lib/cues'
import type { MenuEntry, Recordings } from '../types'
import type { Player } from './player'

export const TALK_URL = 'http://127.0.0.1:1/token/v/talks/Ants.mp3'

export const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 83_000, to: 85_000 },
]

export const TRANSCRIBE_ENTRY: MenuEntry = {
  pluginId: 'md.test.transcription',
  pluginName: 'Transcription',
  command: 'make-text',
  title: 'Transcribe',
  icon: 'audio-lines',
  versionId: '',
  isLive: false,
}

export const DELETE_ENTRY: MenuEntry = {
  pluginId: 'md.test.transcription',
  pluginName: 'Transcription',
  command: 'delete-transcript',
  title: 'Delete transcript',
  icon: '',
  versionId: 'md.test.transcription/abc',
  isLive: true,
}

export const PROOFREAD_ENTRY: MenuEntry = {
  pluginId: 'md.test.proofreading',
  pluginName: 'Proofreading',
  command: 'correct-text',
  title: 'Proofread',
  icon: 'spell-check',
  versionId: 'md.test.transcription/abc',
  isLive: true,
}

/**
 * A recording that tests modify as the application would. `setWorking`
 * reports that work on it began or ended, as the window's tasks do.
 */
export function createRecordings(
  cues: readonly Cue[] = CUES,
  menu: readonly MenuEntry[] = [DELETE_ENTRY],
) {
  const state = {
    cues,
    menu,
    reads: 0,
    writes: [] as (readonly Cue[])[],
    runs: [] as MenuEntry[],
    cueTimes: new Map<number, number>(),
    readError: null as Error | null,
    onWork: null as ((isWorking: boolean) => void) | null,
    isWatched: false,
  }
  const recordings: Recordings = {
    getSource: async () => ({ mediaUrl: TALK_URL, mediaType: 'audio/mpeg' }),
    readTranscript: async () => {
      state.reads++
      if (state.readError) throw state.readError
      return state.cues
    },
    writeTranscript: async (_, next) => void state.writes.push(next),
    findCueTime: async (_, span) => state.cueTimes.get(span.start) ?? null,
    getMenu: async () => state.menu,
    runMenuEntry: async (_, entry) => void state.runs.push(entry),
    watchWork: (_, onChange) => {
      state.onWork = onChange
      state.isWatched = true
      return () => void (state.isWatched = false)
    },
  }
  return { recordings, state }
}

/** The one player the window has, faked: nothing here makes a sound. */
export function createPlayer() {
  const url = ref('')
  const position = ref(0)
  const seeks: [string, number][] = []
  const player: Player = {
    url,
    position,
    duration: ref(0),
    isPlaying: ref(false),
    error: ref(''),
    load: (next) => void (url.value = next),
    play: (next) => void (url.value = next),
    pause: () => {},
    seek: (next, ms) => {
      url.value = next
      position.value = ms
      seeks.push([next, ms])
    },
    getDuration: async () => 0,
  }
  return { player, seeks }
}

/** Resolves once every pending request has been answered. */
export const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

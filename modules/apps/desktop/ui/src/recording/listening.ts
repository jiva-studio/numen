/**
 * One recording as its tab hears it: where the player stands in it, the words
 * heard in it, and which of them is being said now.
 *
 * Apart from the template the way `reading.ts` is: where the player is sent,
 * which cue that lands in, and when the words are asked for again are
 * decisions, and a test asks them without a browser.
 */
import type { Run } from '../core'
import { computed, ref } from 'vue'

/** One stretch of speech: what was said, and the milliseconds it spans. */
export interface Cue {
  readonly text: string
  readonly from: number
  readonly to: number
}

/**
 * What a recording is: how long it runs, and how far the words written down
 * reach, both in milliseconds.
 *
 * A recording nothing has listened to reaches nowhere, and the player it is
 * loaded into is what then says how long it runs.
 */
export interface Listened {
  readonly length: number
  readonly heard: number
}

/** Everything a recording tab asks of the application. */
export interface Recordings {
  /** How long the recording runs, and how much of it has been written down. */
  listened(path: string): Promise<Listened>
  /** Where the recording's own bytes are played from, as an address for a player. */
  media(path: string): string
  /** The words heard in the recording, in the order they were spoken. */
  cues(path: string): Promise<readonly Cue[]>
  /**
   * The millisecond a run of the words written down is played from, and nothing
   * where no cue holds it.
   */
  plays(path: string, run: Run): Promise<number | null>
}

/** The player one recording is heard through, once its tab is drawn. */
export interface Player {
  /** Play from a millisecond of the recording. */
  seek(ms: number): void
}

/** What a container of a recording is played as. */
const SOUNDS: Record<string, string> = {
  mp3: 'audio/mpeg',
  wav: 'audio/wav',
  flac: 'audio/flac',
}

/**
 * Whether this window can play a kind of sound.
 *
 * A build whose media backend is missing answers no to every kind, and a player
 * made for one takes the window down with it. The answer is the window's and
 * not one tab's, so it is asked once however many recordings are open.
 */
const asked = new Map<string, boolean>()

/** Answers is what says whether a kind of sound can be played. A test says. */
export type Answers = (type: string) => boolean

let answers: Answers = (type) => {
  try {
    return document.createElement('audio').canPlayType(type) !== ''
  } catch {
    return false
  }
}

/** Asking puts a different answer in front of the window's own. */
export function asking(said: Answers) {
  answers = said
  asked.clear()
}

/** Playable is whether a recording at a path can be played in this window. */
export function playable(path: string): boolean {
  const type = SOUNDS[path.split('.').pop()?.toLowerCase() ?? '']
  if (!type) return false
  const held = asked.get(type)
  if (held !== undefined) return held
  const can = answers(type)
  asked.set(type, can)
  return can
}

/** A millisecond written out as a person reads a clock. */
export const timed = (ms: number): string => {
  const whole = Math.max(0, Math.floor(ms / 1000))
  const hours = Math.floor(whole / 3600)
  const minutes = Math.floor(whole / 60) % 60
  const seconds = `${whole % 60}`.padStart(2, '0')
  if (hours === 0) return `${minutes}:${seconds}`
  return `${hours}:${`${minutes}`.padStart(2, '0')}:${seconds}`
}

/**
 * The cue being said at a millisecond, and the last one said where a silence
 * stands there. Nothing until the first cue begins.
 */
const holding = (cues: readonly Cue[], ms: number): number => {
  for (let at = cues.length - 1; at >= 0; at--) {
    if (cues[at]!.from <= ms) return at
  }
  return -1
}

export type Listening = ReturnType<typeof listening>

export function listening(recordings: Recordings, path: string) {
  /** Where the recording's own bytes are played from. */
  const address = recordings.media(path)
  /** The words heard in the recording, in the order they were spoken. */
  const cues = ref<readonly Cue[]>([])
  /** How long the recording runs, as the application last said. */
  const length = ref(0)
  /** How much of it has been written down, in milliseconds. */
  const heard = ref(0)
  /** Where the player stands, in milliseconds. */
  const now = ref(0)
  /** Whether something is writing down what this recording says. */
  const working = ref(false)
  /** What this recording could not do, in words the tab puts up for it. */
  const trouble = ref('')

  /** Which cue is being said now, and nothing where none has begun. */
  const current = computed(() => holding(cues.value, now.value))

  /** Whether the tab this recording stands in is still open. */
  let open = true

  /** The player this recording is loaded into, once its tab is drawn. */
  let player: Player | null = null

  /** A moment gone to before there was a player, played from once there is one. */
  let wanted = -1

  /**
   * What the recording is and what has been heard in it. A build that cannot
   * read a transcript says so where the words would stand, and the recording
   * still plays.
   */
  const hear = async () => {
    try {
      const said = await recordings.listened(path)
      if (!open) return
      length.value = said.length
      heard.value = said.heard
      cues.value = await recordings.cues(path)
      if (!open) return
      trouble.value = ''
    } catch (error) {
      if (!open) return
      trouble.value = String(error)
    }
  }

  /** What the recording is, asked for as its tab opens. */
  const opened = hear()

  /** The moment the person went to. Before the beginning is the beginning. */
  const go = (ms: number) => {
    if (!open) return
    now.value = Math.max(0, Math.round(ms))
    if (player) return player.seek(now.value)
    wanted = now.value
  }

  /** The player moved, of itself or under the person's hand. */
  const moved = (ms: number) => {
    if (!open) return
    now.value = Math.max(0, ms)
  }

  /** The tab was drawn, and this is the player it drew. */
  const plays = (into: Player | null) => {
    player = into
    if (!player || wanted < 0) return
    player.seek(wanted)
    wanted = -1
  }

  /**
   * Work on this recording, as the application last reported it. The words are
   * asked for again while a run is going and once more when it stops.
   */
  const ticks = (running: boolean) => {
    if (!open || (!running && !working.value)) return
    working.value = running
    void hear()
  }

  /**
   * Stretches of the words written down reached: the player is sent to the
   * moment the first of them was spoken at. A stretch no cue holds leaves the
   * player where it stands.
   */
  const reach = async (...runs: readonly Run[]) => {
    await opened
    if (!open || runs.length === 0) return
    try {
      const ms = await recordings.plays(path, runs[0]!)
      if (!open || ms === null) return
      go(ms)
    } catch (error) {
      if (!open) return
      trouble.value = String(error)
    }
  }

  /** The tab has closed: nothing is asked for again and nothing is played. */
  const close = () => {
    open = false
    player = null
    cues.value = []
  }

  return {
    path,
    address,
    /** Whether this window can play the recording at all. */
    playable: playable(path),
    cues,
    length,
    heard,
    now,
    current,
    working,
    trouble,
    go,
    moved,
    plays,
    ticks,
    reach,
    close,
  }
}

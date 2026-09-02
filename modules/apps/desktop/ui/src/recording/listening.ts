/**
 * One recording as its tab hears it: where the player stands in it, the words
 * heard in it, which of them is being said now, and the words as a person
 * edits them.
 *
 * Apart from the template the way `reading.ts` is: where the player is sent,
 * which cue that lands in, and when the words are asked for again are
 * decisions, and a test asks them without a browser.
 */
import type { Run } from '../core'
import { computed, ref } from 'vue'
import { clock } from '@numen/ui'
import { cued, same, spanning, spoken } from './cueing'
import { player, type Player } from './playing'
import { WORDS } from './words'

/** One stretch of speech: what was said, and the milliseconds it spans. */
export interface Cue {
  readonly text: string
  readonly from: number
  readonly to: number
}

/** What was heard in a recording, and whether it may be written over. */
export interface Spoken {
  readonly cues: readonly Cue[]
  /** False while a run listening to the recording holds it. */
  readonly editable: boolean
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
  /** Where the recording is played from, as the application answers it. */
  readonly media: string
  /** What it is played as. The application says: what counts as a recording is
   * its to decide. */
  readonly type: string
}

/** Everything a recording tab asks of the application. */
export interface Recordings {
  /** How long the recording runs, and how much of it has been written down. */
  listened(path: string): Promise<Listened>
  /** The words heard in the recording, in the order they were spoken. */
  cues(path: string): Promise<Spoken>
  /** The words as a person has edited them, kept against the recording. */
  writes(path: string, cues: readonly Cue[]): Promise<void>
  /**
   * The millisecond a run of the words written down is played from, and nothing
   * where no cue holds it.
   */
  plays(path: string, run: Run): Promise<number | null>
}

// Whether this window can play a kind of sound. The answer is the window's and
// is asked once, however many recordings are open.
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

/** Playable is whether this window can play a recording of a media type. */
export function plays(type: string): boolean {
  if (!type) return false
  const held = asked.get(type)
  if (held !== undefined) return held
  const can = answers(type)
  asked.set(type, can)
  return can
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

/** How long the words have to have been still before they are written. */
export const QUIET = 800

export function listening(
  recordings: Recordings,
  path: string,
  through: Player = player,
  quiet = QUIET,
) {
  /** Where the recording's own bytes are played from, once it is asked. */
  const address = ref('')
  /** The words heard in the recording, in the order they were spoken. */
  const cues = ref<readonly Cue[]>([])
  /** The words as the editor shows them, one cue to a line. */
  const prose = ref('')
  /** Whether the transcript may be written over now. */
  const editable = ref(true)
  /** Whether the view keeps the line being said in sight. */
  const following = ref(true)
  /** How long the recording runs, as the application last said. */
  const length = ref(0)
  /** How much of it has been written down, in milliseconds. */
  const heard = ref(0)
  /** What the recording is played as, as the application answers it. */
  const type = ref('')
  /** Whether the recording the player holds is this one. */
  const held = computed(() => address.value !== '' && through.address.value === address.value)

  /**
   * How long the recording runs. The application says, and the recording
   * itself says where it is loaded and knows better.
   */
  const runs = computed(() => Math.max(length.value, held.value ? through.length.value : 0))

  /**
   * Where the player stands in this recording, in milliseconds.
   *
   * The window plays one recording at a time, so one the player is not holding
   * stands at its beginning until somebody plays it.
   */
  const now = computed(() => (held.value ? through.at.value : 0))

  /** Whether this recording is the one playing. */
  const playing = computed(() => held.value && through.playing.value)
  /** Whether something is writing down what this recording says. */
  const working = ref(false)
  /** What this recording could not do, in words the tab puts up for it. */
  const trouble = ref('')
  /** What the player could not do, while this is the recording it holds. */
  const broken = computed(() => (held.value ? through.failed.value : ''))

  /**
   * The lines on screen against the milliseconds they cover. The lines are
   * what a person edits, and these follow them until the file is written.
   *
   * A recording nothing was heard in and nothing was typed into has no lines
   * at all, and the tab says so where they would stand.
   */
  const spans = computed(() =>
    cues.value.length === 0 && prose.value === '' ? [] : spanning(cues.value, prose.value),
  )

  /** Which line is being said now, and nothing where none has begun. */
  const current = computed(() => holding(spans.value, now.value))

  /** Whether the tab this recording stands in is still open. */
  let open = true

  /**
   * A moment gone to before the recording knew its own address, played from
   * once it does. A hit in the words opens a tab and asks for a moment in the
   * same breath.
   */
  let wanted = -1

  /**
   * asking counts the times the words have been asked for, and answered holds
   * the last count to have landed. A run being written down is asked about
   * again while it goes, and two answers may arrive in either order; the older
   * of them carries fewer words, and writing it down would take words off the
   * screen a person is reading.
   */
  let asking = 0
  let answered = 0

  /** Whether what is on screen has still to reach the file. */
  let owed = false
  /** A write of the words that has not answered yet. */
  let writing = false
  /** The wait the typing is being let settle over. */
  let settling: ReturnType<typeof setTimeout> | undefined

  /**
   * What the recording is and what has been heard in it. A build that cannot
   * read a transcript says so where the words would stand, and the recording
   * still plays.
   */
  const hear = async () => {
    const count = ++asking
    try {
      const said = await recordings.listened(path)
      if (!open || count < answered) return
      answered = count
      length.value = said.length
      heard.value = said.heard
      address.value = said.media
      type.value = said.type
      // A moment asked for before the recording knew where its bytes are.
      if (wanted >= 0 && address.value) {
        const at = wanted
        wanted = -1
        through.seek(address.value, at)
      }
      // The player holding nothing takes this recording, so the controls read
      // how long it runs before anybody presses play. One already in the
      // player is left where it is.
      if (address.value && through.address.value === '') through.load(address.value)

      const spoke = await recordings.cues(path)
      if (!open || count < answered) return
      answered = count
      cues.value = spoke.cues
      editable.value = spoke.editable
      // Words the person has typed and not yet had written stay on screen.
      if (!owed) prose.value = spoken(spoke.cues)
      trouble.value = ''
    } catch (error) {
      if (!open || count < answered) return
      answered = count
      trouble.value = String(error)
    }
  }

  /** What the recording is, asked for as its tab opens. */
  const opened = hear()

  /** The moment the person went to. Before the beginning is the beginning. */
  const go = (ms: number) => {
    if (!open) return
    const at = Math.max(0, Math.round(ms))
    if (!address.value) return void (wanted = at)
    through.seek(address.value, at)
  }

  /** Play this recording, taking the sound from whatever else held it. */
  const play = () => {
    if (!open || !address.value) return
    through.play(address.value)
  }

  /** Stop it, while it is this recording that is playing. */
  const pause = () => {
    if (playing.value) through.pause()
  }

  /** The line a person asked for, counted from the first line on screen. */
  const goes = (line: number) => {
    const span = spans.value[line]
    if (span) go(span.from)
  }

  /** Whether the view keeps the line being said in sight. */
  const follows = (on: boolean) => {
    following.value = on
  }

  /** The person typed. The words are written once they have been still. */
  const typed = (body: string) => {
    if (!open || body === prose.value) return
    prose.value = body
    owed = true
    clearTimeout(settling)
    settling = setTimeout(() => void keep(), quiet)
  }

  /**
   * The words as they now read, kept against the recording. A write that is
   * refused leaves them owed, so the next stillness offers them again.
   */
  const keep = async () => {
    clearTimeout(settling)
    settling = undefined
    if (!open || !owed || writing || !editable.value) return
    const body = prose.value
    const next = cued(cues.value, body)
    owed = false
    // A transcript written down is a transcript a person owns, and a
    // proofreader leaves it alone. Only words that changed are written.
    if (same(next, cues.value)) return
    writing = true
    try {
      await recordings.writes(path, next)
      if (!open) return
      cues.value = next
      trouble.value = ''
    } catch (error) {
      if (!open) return
      owed = true
      trouble.value = String(error)
    } finally {
      writing = false
      // Typing that landed while the write was in the air is still owed.
      if (open && prose.value !== body) {
        owed = true
        settling = setTimeout(() => void keep(), quiet)
      }
    }
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

  /**
   * The tab has closed: what the person typed reaches the file, nothing is
   * asked for again, and the recording stops where the player stands in it.
   *
   * Two tabs may stand on one recording, and closing either of them stops it.
   */
  const close = () => {
    void keep()
    pause()
    open = false
    cues.value = []
    prose.value = ''
  }

  /**
   * The moment each line was said at, on a clock, as the editor's gutter draws
   * them. These follow the words alone, so a transcript of any length is
   * written out once and left alone while the recording plays.
   */
  const times = computed(() => spans.value.map((cue) => clock(cue.from)))

  /** Whether this window can play a recording of this kind at all. */
  const playable = computed(() => address.value !== '' && plays(type.value))

  /** What the tab says where the words would stand, and nothing where they do. */
  const note = computed(() => {
    if (times.value.length) return ''
    if (working.value) return WORDS.transcribing
    return WORDS.silence
  })

  return {
    path,
    address,
    playable,
    times,
    note,
    cues,
    prose,
    editable,
    following,
    length,
    runs,
    heard,
    now,
    current,
    working,
    trouble,
    broken,
    go,
    goes,
    follows,
    typed,
    keep,
    playing,
    play,
    pause,
    ticks,
    reach,
    close,
  }
}

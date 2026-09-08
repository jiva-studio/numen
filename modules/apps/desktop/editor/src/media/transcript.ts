/**
 * One recording as its tab reads it: where the player stands in it, its
 * transcript, which cue is being said now, and the words as a person edits
 * them.
 */
import type { Stretch } from '../shared/core'
import { computed, ref, shallowRef } from 'vue'
import { clock } from '@numen/ui'
import { troubleWords } from '@numen/wire'
import { answerGuard as latest } from '../shared/questions'
import { cued, same, spanning, spoken, type Cue } from './cues'
import { playable as canPlay, player, type MediaTypeProbe, type Player } from './player'
import { WORDS } from './words'

/** The player a tab drew for itself, which a moment chosen in the words seeks. */
export interface TabPlayer {
  seeks(ms: number): void
}

/** The text fetched or heard, and whether it may be written over. */
export interface Transcript {
  readonly cues: readonly Cue[]
  /**
   * The text where nothing timed it: the prose a page is written around. It
   * comes instead of the cues, never beside them.
   */
  readonly prose: string
  /** False while a run writing the transcript holds it. */
  readonly editable: boolean
}

/**
 * What a file is, for whatever plays it: how long it runs, in milliseconds,
 * and where its bytes come from.
 *
 * A recording nothing has listened to reaches nowhere, and the player it is
 * loaded into is what then says how long it runs.
 */
export interface RecordingSummary {
  readonly duration: number
  /** Where it is played from, as the application answers it. */
  readonly mediaUrl: string
  /**
   * What it is played as. The application says: what counts as a recording is
   * its to decide, and a url with no copy on this disk is a page to frame.
   */
  readonly mediaType: string
  /** The web address a url points at, and nothing on every other source. */
  readonly url: string
}

/** Everything a recording tab asks of the application. */
export interface Recordings {
  /** How long the recording runs, and how much of it has been written down. */
  listened(path: string): Promise<RecordingSummary>
  /** The words heard in the recording, in the order they were spoken. */
  cues(path: string): Promise<Transcript>
  /** The words as a person has edited them, kept against the recording. */
  writes(path: string, cues: readonly Cue[]): Promise<void>
  /**
   * The millisecond a stretch of the words written down is played from, and
   * nothing where no cue holds it.
   */
  plays(path: string, stretch: Stretch): Promise<number | null>
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

export type TranscriptState = ReturnType<typeof transcript>

/** How long the words have to have been still before they are written. */
export const QUIET = 800

/** What the window hands one recording tab, beside the application and the path. */
export interface TranscriptOptions {
  /** The player the sound comes out of, which every recording of a window shares. */
  through?: Player
  /** How long the typing settles for before the words are written. */
  quiet?: number
  /** Whether this window can play a kind of sound. */
  plays?: MediaTypeProbe
}

/** What a url with no copy on this disk is played as: a page, read in a frame. */
const PAGE = 'text/html'

export function transcript(recordings: Recordings, path: string, how: TranscriptOptions = {}) {
  const through = how.through ?? player
  const quiet = how.quiet ?? QUIET
  const plays = how.plays ?? canPlay()
  /** Where the recording's own bytes are played from, once it is asked. */
  const address = ref('')
  /** The words heard in the recording, in the order they were spoken. */
  const cues = shallowRef<readonly Cue[]>([])
  /** The words as the editor shows them, one cue to a line. */
  const prose = ref('')
  /** Whether the transcript may be written over now. */
  const editable = ref(true)
  /** Whether the view keeps the line being said in sight. */
  const following = ref(true)
  /** Whether a person has been typing too recently for the view to move. */
  const typing = ref(false)
  /** How long the recording runs, as the application last said. */
  const duration = ref(0)
  /** What the recording is played as, as the application answers it. */
  const type = ref('')
  /** The web address a url points at, and nothing on every other source. */
  const points = ref('')
  /** Whether the recording the player holds is this one. */
  const held = computed(() => address.value !== '' && through.address.value === address.value)

  /**
   * How long the recording runs. The application says, and the recording
   * itself says where it is loaded and knows better.
   */
  const runs = computed(() => Math.max(duration.value, held.value ? through.duration.value : 0))

  /**
   * Where the player stands in this recording, in milliseconds.
   *
   * The window plays one recording at a time, so one the player is not holding
   * stands at its beginning until somebody plays it.
   */
  const now = computed(() => {
    if (frame.value) return Math.max(0, framed.value)
    return held.value ? through.at.value : 0
  })

  /**
   * The player drawn in the tab, which a url has and a recording does not: what
   * a url points at plays where it is drawn, not through the one the window
   * plays sound with.
   */
  const frame = shallowRef<TabPlayer | null>(null)
  /** Where that player stands, and nothing before it has said. */
  const framed = ref(-1)

  /** The tab hands over the player it drew, and says where it stands. */
  const playsIn = (player: TabPlayer | null) => {
    frame.value = player
    framed.value = -1
  }
  const reached = (ms: number) => {
    framed.value = ms
  }

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

  /**
   * Whether the text carries the times each stretch of it was said at. A page
   * is prose and carries none: there is nothing to seek and no line being said.
   */
  const timed = computed(() => cues.value.length > 0)

  /** Whether anything at all has been fetched or heard here. */
  const written = computed(() => timed.value || prose.value !== '')

  /** Which line is being said now, and nothing where none has begun. */
  const current = computed(() =>
    frame.value && framed.value < 0 ? -1 : holding(spans.value, now.value),
  )

  /** Whether the tab this recording stands in is still open. */
  let open = true

  /**
   * A moment gone to before the recording knew its own address, played from
   * once it does. A hit in the words opens a tab and asks for a moment in the
   * same breath.
   */
  let wanted = -1

  /**
   * A run being written down is asked about again while it goes, and two
   * answers may arrive in either order. The older of them carries fewer words.
   */
  const asks = latest()

  /** Whether what is on screen has still to reach the file. */
  let owed = false
  /** A write of the words that has not answered yet. */
  let writing = false
  /** The wait the typing is being let settle over. */
  let settling: ReturnType<typeof setTimeout> | undefined
  /** The wait after which the view may go after the words again. */
  let stilling: ReturnType<typeof setTimeout> | undefined

  /**
   * What the recording is and what has been written down of it. A build that
   * cannot read a transcript says so where the words would stand, and the
   * recording still plays.
   */
  const reads = async () => {
    const mine = asks.ask()
    try {
      const said = await recordings.listened(path)
      if (!mine.lands()) return
      duration.value = said.duration
      address.value = said.mediaUrl
      type.value = said.mediaType
      points.value = said.url
      // A moment asked for before the recording knew where its bytes are.
      if (wanted >= 0 && address.value) {
        const at = wanted
        wanted = -1
        through.seek(address.value, at)
      }
      // The player holding nothing takes this recording, so the controls read
      // how long it runs before anybody presses play. One already in the
      // player is left where it is, and a url plays where it is drawn.
      if (!points.value && address.value && through.address.value === '') {
        through.load(address.value)
      }

      const spoke = await recordings.cues(path)
      if (!mine.lands()) return
      cues.value = spoke.cues
      editable.value = spoke.editable
      // Words the person has typed and not yet had written stay on screen.
      if (!owed) prose.value = spoke.cues.length ? spoken(spoke.cues) : spoke.prose
      trouble.value = ''
    } catch (error) {
      if (!mine.lands()) return
      trouble.value = troubleWords(error)
    }
  }

  /** What the recording is, asked for as its tab opens. */
  const opened = reads()

  /** The moment the person went to. Before the beginning is the beginning. */
  const go = (ms: number) => {
    if (!open) return
    const at = Math.max(0, Math.round(ms))
    if (frame.value) return frame.value.seeks(at)
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
    // An edit that adds or takes away a line moves which line is being said,
    // and the view does not go after a line a person moved under their own
    // hands.
    typing.value = true
    clearTimeout(stilling)
    stilling = setTimeout(() => void (typing.value = false), quiet)
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
      trouble.value = troubleWords(error)
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
   * The words are asked for again, and what stands on screen is whatever comes
   * back. An edit still waiting to be written goes: it was of words that are no
   * longer the ones the recording has.
   */
  const again = () => {
    clearTimeout(settling)
    settling = undefined
    owed = false
    void reads()
  }

  /**
   * Work on this recording, as the application last reported it. The words are
   * asked for again while a run is going and once more when it stops.
   */
  const ticks = (running: boolean) => {
    if (!open || (!running && !working.value)) return
    working.value = running
    void reads()
  }

  /**
   * Stretches of the words written down reached: the player is sent to the
   * moment the first of them was spoken at. A stretch no cue holds leaves the
   * player where it stands.
   */
  const reach = async (...stretches: readonly Stretch[]) => {
    await opened
    if (!open || stretches.length === 0) return
    try {
      const ms = await recordings.plays(path, stretches[0]!)
      if (!open || ms === null) return
      go(ms)
    } catch (error) {
      if (!open) return
      trouble.value = troubleWords(error)
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
    asks.close()
    clearTimeout(stilling)
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

  /**
   * Whether what is at the address is a page, which is read in a frame rather
   * than played. A url with no copy on this disk is one.
   */
  const framing = computed(() => address.value !== '' && type.value === PAGE)

  /** How far into the recording the words written down reach, in milliseconds. */
  const transcribedDuration = computed(() => cues.value.at(-1)?.to ?? 0)

  /** What the tab says where the words would stand, and nothing where they do. */
  const note = computed(() => {
    if (times.value.length) return ''
    if (working.value) return WORDS.transcribing
    return points.value ? WORDS.unfetched : WORDS.silence
  })

  return {
    path,
    address,
    points,
    framing,
    playable,
    times,
    spans,
    note,
    cues,
    prose,
    editable,
    following,
    typing,
    duration,
    runs,
    transcribedDuration,
    now,
    current,
    timed,
    written,
    playsIn,
    reached,
    working,
    trouble,
    broken,
    go,
    goes,
    follows,
    typed,
    keep,
    again,
    playing,
    play,
    pause,
    ticks,
    reach,
    close,
  }
}

/**
 * One recording as its tab reads it: where the player stands in it, its
 * transcript, which cue is being said now, and the words as a person edits
 * them.
 */
import type { Span } from '@/shared/span'
import { computed, ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { createAnswerGuard as latest } from '@/shared/questions'
import { findCueAt, getText, type Cue } from '../lib/cues'
import { useTranscriptDraft } from './draft'
import { useTranscriptLines } from './lines'
import { usePlayback } from './playback'
import { createMediaTypeProbe, player, type MediaTypeProbe, type Player } from './player'
import type { Recordings } from '../types'
import { WORDS } from '../words'

export type { TabPlayer } from './playback'

export type TranscriptState = ReturnType<typeof useTranscript>

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

export function useTranscript(recordings: Recordings, path: string, how: TranscriptOptions = {}) {
  const through = how.through ?? player
  const quiet = how.quiet ?? QUIET
  const plays = how.plays ?? createMediaTypeProbe()

  /** The words heard in the recording, in the order they were spoken. */
  const cues = shallowRef<readonly Cue[]>([])
  /** Whether the transcript may be written over now. */
  const isEditable = ref(true)
  /** Whether the view keeps the line being said in sight. */
  const following = ref(true)
  /** Whether something is writing down what this recording says. */
  const isWorking = ref(false)
  /** What this recording could not do, in words the tab puts up for it. */
  const error = ref('')

  /**
   * Whether the first reading of the recording is still on its way. Until it
   * lands nothing is known: no words is not the same as no words yet, and a tab
   * that cannot tell them apart offers to write down what is already written.
   */
  const isLoading = ref(true)

  /** Whether the tab this recording stands in is still open. */
  let open = true
  const isOpen = () => open

  const {
    url,
    duration,
    points,
    runs,
    now,
    hasTime,
    isPlaying,
    broken,
    playable,
    framing,
    setFramePlayer,
    setFrameTime,
    setSource,
    go,
    play,
    pause,
  } = usePlayback(through, plays, isOpen)

  const { prose, typing, keep, setProse, setText, drop, stop } = useTranscriptDraft({
    cues,
    isEditable,
    error,
    quiet,
    isOpen,
    write: (next) => recordings.writeTranscript(path, next),
  })

  const { spans, timed, written, times, transcribedDuration } = useTranscriptLines(cues, prose)

  /** Which line is being said now, and nothing where none has begun. */
  const current = computed(() => (hasTime.value ? findCueAt(spans.value, now.value) : -1))

  /**
   * A run being written down is asked about again while it goes, and two
   * answers may arrive in either order. The older of them carries fewer words.
   */
  const asks = latest()

  /**
   * What the recording is and what has been written down of it. A build that
   * cannot read a transcript says so where the words would stand, and the
   * recording still plays.
   */
  const readRecording = async () => {
    const mine = asks.ask()
    try {
      const said = await recordings.getSummary(path)
      if (!mine.claim()) return
      setSource(said)

      // What the file carries says which text to read. A url publishing words
      // against a clock carries a transcript; every other page carries prose.
      const carried = await recordings.getTaskStates(path)
      if (!mine.claim()) return
      const spoke = await (carried.transcript
        ? recordings.readTranscript(path)
        : recordings.readArticle(path))
      if (!mine.claim()) return
      cues.value = spoke.cues
      isEditable.value = spoke.isEditable
      setText(spoke.cues.length ? getText(spoke.cues) : spoke.prose)
      error.value = ''
    } catch (thrown) {
      if (!mine.claim()) return
      error.value = formatErrorMessage(thrown)
    } finally {
      if (mine.claim()) isLoading.value = false
    }
  }

  /** What the recording is, asked for as its tab opens. */
  const opened = readRecording()

  /** The line a person asked for, counted from the first line on screen. */
  const goToLine = (line: number) => {
    const span = spans.value[line]
    if (span) go(span.from)
  }

  /** Whether the view keeps the line being said in sight. */
  const setFollowing = (on: boolean) => {
    following.value = on
  }

  /** The words are asked for again, and what stands on screen is whatever comes back. */
  const reload = () => {
    drop()
    void readRecording()
  }

  /**
   * Work on this recording, as the application last reported it. The words are
   * asked for again while a run is going and once more when it stops.
   */
  const setWorking = (next: boolean) => {
    if (!open || (!next && !isWorking.value)) return
    isWorking.value = next
    void readRecording()
  }

  /**
   * Spanes of the words written down reached: the player is sent to the
   * moment the first of them was spoken at. A span no cue holds leaves the
   * player where it stands.
   */
  const reach = async (...spans: readonly Span[]) => {
    await opened
    if (!open || spans.length === 0) return
    try {
      const ms = await recordings.findCueTime(path, spans[0]!)
      if (!open || ms === null) return
      go(ms)
    } catch (thrown) {
      if (!open) return
      error.value = formatErrorMessage(thrown)
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
    stop()
    cues.value = []
  }

  /** What the tab says where the words would stand, and nothing where they do. */
  const note = computed(() => {
    if (times.value.length) return ''
    if (isWorking.value) return WORDS.transcribing
    return points.value ? WORDS.unfetched : WORDS.silence
  })

  return {
    path,
    url,
    points,
    framing,
    playable,
    times,
    spans,
    note,
    cues,
    prose,
    isEditable,
    following,
    typing,
    duration,
    runs,
    transcribedDuration,
    now,
    current,
    timed,
    written,
    setFramePlayer,
    setFrameTime,
    isWorking,
    isLoading,
    error,
    broken,
    go,
    goToLine,
    setFollowing,
    setProse,
    keep,
    reload,
    isPlaying,
    play,
    pause,
    setWorking,
    reach,
    close,
  }
}

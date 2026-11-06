/**
 * The state of one recording tab: playback position, transcript cues, the current cue,
 * the edited text, and whether its text can be created.
 */
import { computed, ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { findCueAt, getText, type Cue } from '../lib/cues'
import { useTranscriptDraft } from './draft'
import { useTranscriptLines } from './lines'
import { usePlayback } from './playback'
import { createMediaTypeProbe, player, type MediaTypeProbe, type Player } from './player'
import { recordings as servedRecordings } from '../api/wire'
import type { Recordings, TextRange } from '../types'
import { WORDS } from '../words'

export type TranscriptState = ReturnType<typeof useTranscript>

/** How long the words have to have been still before they are written. */
export const WRITE_IDLE_MS = 800

/** What the window hands one recording tab, beside the application and the path. */
export interface TranscriptOptions {
  /** The player the sound comes out of, which every recording of a window shares. */
  audioPlayer?: Player
  /** How long the typing settles for before the words are written. */
  idleMs?: number
  /** Whether this window can play a media type. */
  plays?: MediaTypeProbe
}

/** Request ordering: only the latest request may apply its result. */
function createRequestGuard() {
  let latest = 0
  let isClosed = false
  return {
    begin: () => {
      const mine = ++latest
      return { isLatest: () => !isClosed && mine === latest }
    },
    close: () => {
      isClosed = true
    },
  }
}

export function useTranscript(
  recordingsParam: Recordings | undefined,
  path: string,
  how: TranscriptOptions = {},
) {
  const recordings = recordingsParam ?? servedRecordings
  const audioPlayer = how.audioPlayer ?? player
  const idleMs = how.idleMs ?? WRITE_IDLE_MS
  const plays = how.plays ?? createMediaTypeProbe()

  let isOpen = true
  const checkOpen = () => isOpen

  const cues = shallowRef<readonly Cue[]>([])
  const canCreateText = ref(false)
  const isWorking = ref(false)
  const isLoading = ref(true)
  const error = ref('')
  const following = ref(true)
  const isEditable = computed(() => !isWorking.value)
  const requests = createRequestGuard()

  const playback = usePlayback(audioPlayer, plays, checkOpen)
  const draft = useTranscriptDraft({
    cues,
    isEditable,
    error,
    idleMs,
    isOpen: checkOpen,
    write: (next) => recordings.writeTranscript(path, next),
  })
  const lines = useTranscriptLines(cues, draft.prose)

  const readRecording = async () => {
    const mine = requests.begin()
    try {
      const [source, transcriptCues, isCreatable] = await Promise.all([
        recordings.getSource(path),
        recordings.readTranscript(path),
        recordings.canCreateText(path),
      ])
      if (!mine.isLatest()) return
      playback.setSource(source)
      cues.value = transcriptCues
      canCreateText.value = isCreatable
      draft.setText(getText(transcriptCues))
      error.value = ''
    } catch (err) {
      if (!mine.isLatest()) return
      error.value = formatErrorMessage(err)
    } finally {
      if (mine.isLatest()) isLoading.value = false
    }
  }

  const firstRead = readRecording()

  /** Shows the failure of an operation that ran in the background. */
  const reportFailure = (err: unknown) => {
    if (isOpen) error.value = formatErrorMessage(err)
  }

  /** Work on the recording began or ended: the words are read again. */
  const setWorking = (next: boolean) => {
    if (!isOpen || (!next && !isWorking.value)) return
    isWorking.value = next
    readRecording().catch(reportFailure)
  }
  const stopWorking = recordings.watchWork(path, setWorking)
  const stopChanges = recordings.watchChanges(path, () => {
    if (isOpen) readRecording().catch(reportFailure)
  })
  const stopWatching = () => {
    stopWorking()
    stopChanges()
  }

  const isSettled = computed(() => !isLoading.value && !isWorking.value)
  const isStartable = computed(() => isSettled.value && !lines.timed.value && canCreateText.value)

  const createText = async () => {
    if (!isOpen) return
    try {
      await recordings.createText(path)
      draft.drop()
      await readRecording()
    } catch (err) {
      if (isOpen) error.value = formatErrorMessage(err)
    }
  }

  const current = computed(() => findCueAt(lines.spans.value, playback.now.value))
  const note = computed(() => {
    if (lines.times.value.length) return ''
    return isWorking.value ? WORDS.transcribing : WORDS.silence
  })

  /** Moves playback to the start time of the first of the spans. */
  const reach = async (...spans: readonly TextRange[]) => {
    await firstRead
    if (!isOpen || spans.length === 0) return
    try {
      const ms = await recordings.findCueTime(path, spans[0]!)
      if (isOpen && ms !== null) playback.controls.go(ms)
    } catch (err) {
      if (isOpen) error.value = formatErrorMessage(err)
    }
  }

  const goToLine = (line: number) => {
    const span = lines.spans.value[line]
    if (span) playback.controls.go(span.from)
  }

  const close = () => {
    draft.keep().catch(reportFailure)
    playback.controls.pause()
    isOpen = false
    stopWatching()
    requests.close()
    cues.value = []
    draft.stop()
  }

  return {
    path,
    reportFailure,
    playback,
    draft,
    lines,
    content: {
      cues,
      isWorking,
      isLoading,
      isEditable,
      error,
      firstRead,
      note,
      current,
      following,
      setFollowing: (isOn: boolean) => {
        following.value = isOn
      },
      setWorking,
      reload: () => {
        draft.drop()
        readRecording().catch(reportFailure)
      },
    },
    options: {
      isStartable,
      start: createText,
    },
    navigation: { reach, goToLine },
    close,
  }
}

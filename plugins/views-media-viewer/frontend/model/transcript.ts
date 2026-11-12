/**
 * The state of one recording tab: playback position, transcript cues, the current cue,
 * the edited text, a live text without times shown as it is, the page that frames the
 * player while there is no media, and whether its text can be created.
 */
import { computed, ref, shallowRef } from 'vue'
import { formatErrorMessage, type Caught } from '@numen/plugin-sdk'
import { findCueAt, getText } from '../lib/cues'
import { useTranscriptDraft } from './draft'
import { useTranscriptLines } from './lines'
import { usePlayback } from './playback'
import { createMediaTypeProbe, player, type MediaTypeProbe, type Player } from './player'
import { recordings as servedRecordings } from '../api/wire'
import type { Recordings, TextRange, Cue } from '../types'
import { WORDS } from '../words'

export type TranscriptState = ReturnType<typeof useTranscript>

/** How long the words have to have been still before they are written. */
const WRITE_IDLE_MS = 800

/** What the window hands one recording tab, beside the application and the path. */
export interface TranscriptOptions {
  audioPlayer?: Player
  idleMs?: number
  plays?: MediaTypeProbe
}

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
  const getOpenState = () => isOpen

  const cues = shallowRef<readonly Cue[]>([])
  /** Whether the live text has no times, such as an article: it is shown as it is and not edited. */
  const isArticle = ref(false)
  const pageUrl = ref('')
  const canCreateText = ref(false)
  const isWorking = ref(false)
  const isLoading = ref(true)
  const error = ref('')
  const following = ref(true)
  const isEditable = computed(() => !isWorking.value && !isArticle.value)
  const requests = createRequestGuard()

  const playback = usePlayback(audioPlayer, plays, getOpenState)
  const draft = useTranscriptDraft({
    cues,
    isEditable,
    error,
    idleMs,
    isOpen: getOpenState,
    write: (next) => recordings.writeTranscript(path, next),
  })
  const lines = useTranscriptLines(cues, draft.prose)

  const readRecording = async () => {
    const mine = requests.begin()
    try {
      const [source, transcriptCues, article, isCreatable] = await Promise.all([
        recordings.getSource(path),
        recordings.readTranscript(path),
        recordings.readArticle(path),
        recordings.canCreateText(path),
      ])
      if (!mine.isLatest()) return
      playback.source.setSource(source)
      pageUrl.value = source.pageUrl
      cues.value = transcriptCues
      isArticle.value = transcriptCues.length === 0 && article !== ''
      canCreateText.value = isCreatable
      draft.setText(isArticle.value ? article : getText(transcriptCues))
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
  const reportFailure = (err: Caught) => {
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
  const stopWatchingTranscript = () => {
    stopWorking()
    stopChanges()
  }

  const isSettled = computed(() => !isLoading.value && !isWorking.value)
  const isStartable = computed(
    () => isSettled.value && !lines.timed.value && !isArticle.value && canCreateText.value,
  )

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

  const current = computed(() => findCueAt(lines.spans.value, playback.progress.now.value))
  const note = computed(() => {
    if (lines.times.value.length) return ''
    return isWorking.value ? WORDS.transcribing : WORDS.silence
  })

  /** Number of seek requests. */
  let seekRequestCount = 0
  /**
   * Seeks playback to the start time of the first of the spans, and returns that time in
   * milliseconds, or null when playback did not seek. A cue time that resolves after a later seek
   * request is dropped.
   */
  const seekToSpans = async (...spans: readonly TextRange[]): Promise<number | null> => {
    const request = ++seekRequestCount
    await firstRead
    if (!isOpen || spans.length === 0) return null
    try {
      const ms = await recordings.findCueTime(path, spans[0]!)
      if (!isOpen || ms === null || request !== seekRequestCount) return null
      playback.controls.go(ms)
      return ms
    } catch (err) {
      if (isOpen) error.value = formatErrorMessage(err)
      return null
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
    stopWatchingTranscript()
    requests.close()
    cues.value = []
    draft.stop()
  }

  return {
    path,
    playback,
    draft,
    lines,
    content: {
      cues,
      isArticle,
      pageUrl,
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
    navigation: { seekToSpans, goToLine },
    lifecycle: { reportFailure, close },
  }
}

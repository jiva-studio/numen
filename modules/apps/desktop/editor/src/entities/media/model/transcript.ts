/**
 * One recording as its tab reads it: where the player stands in it, its
 * transcript, which cue is being said now, and the words as a person edits
 * them.
 */
import type { Span } from '@/shared/span'
import { computed, ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { createAnswerGuard as latest } from '@numen/plugin-sdk'
import { findCueAt, getText, type Cue } from '../lib/cues'
import { useTranscriptDraft } from './draft'
import { useTranscriptLines } from './lines'
import { usePlayback } from './playback'
import { createMediaTypeProbe, player, type MediaTypeProbe, type Player } from './player'
import type { Recordings, RecordingSummary } from '../types'
import { WORDS } from '../words'

export type { TabPlayer } from './playback'

export type TranscriptState = ReturnType<typeof useTranscript>

/** How long the words have to have been still before they are written. */
export const WRITE_IDLE_MS = 800

/** What the window hands one recording tab, beside the application and the path. */
export interface TranscriptOptions {
  /** The player the sound comes out of, which every recording of a window shares. */
  through?: Player
  /** How long the typing settles for before the words are written. */
  quiet?: number
  /** Whether this window can play a kind of sound. */
  plays?: MediaTypeProbe
}

async function loadRecordingContent(
  source: { recordings: Recordings; path: string },
  mine: { claim: () => boolean },
  options: {
    setSource: (summary: RecordingSummary) => void
    setText: (body: string) => void
  },
) {
  const summary = await source.recordings.getSummary(source.path)
  if (!mine.claim()) return null
  options.setSource(summary)
  const carried = await source.recordings.getTaskStates(source.path)
  if (!mine.claim()) return null
  const content = await (carried.transcript
    ? source.recordings.readTranscript(source.path)
    : source.recordings.readArticle(source.path))
  // Stryker disable next-line ConditionalExpression: Discards results when a newer read supersedes.
  if (!mine.claim()) return null
  options.setText(content.cues.length ? getText(content.cues) : content.prose)
  return content
}

function useTranscriptContent(
  recordings: Recordings,
  path: string,
  options: {
    setSource: (summary: RecordingSummary) => void
    setText: (body: string) => void
    drop: () => void
  },
) {
  const cues = shallowRef<readonly Cue[]>([])
  const isEditable = ref(true)
  const isWorking = ref(false)
  const error = ref('')
  const isLoading = ref(true)
  const asks = latest()

  const readRecording = async () => {
    const mine = asks.ask()
    try {
      const content = await loadRecordingContent({ recordings, path }, mine, options)
      // Stryker disable next-line ConditionalExpression: Discards results when a newer read supersedes.
      if (!mine.claim() || !content) return
      cues.value = content.cues
      isEditable.value = content.isEditable
      error.value = ''
    } catch (thrown) {
      if (!mine.claim()) return
      error.value = formatErrorMessage(thrown)
    } finally {
      if (mine.claim()) isLoading.value = false
    }
  }

  const opened = readRecording()

  return {
    cues,
    isEditable,
    isWorking,
    error,
    isLoading,
    opened,
    reload: () => {
      options.drop()
      void readRecording()
    },
    setWorking: (next: boolean, isOpen: boolean) => {
      if (!isOpen || (!next && !isWorking.value)) return
      isWorking.value = next
      void readRecording()
    },
    close: () => {
      asks.close()
      cues.value = []
    },
  }
}

function createTranscriptNavigation(deps: {
  recordings: Recordings
  path: string
  content: ReturnType<typeof useTranscriptContent>
  playback: ReturnType<typeof usePlayback>
  lines: ReturnType<typeof useTranscriptLines>
  isOpen: () => boolean
}) {
  const { recordings, path, content, playback, lines, isOpen } = deps
  const reach = async (...spans: readonly Span[]) => {
    await content.opened
    if (!isOpen() || spans.length === 0) return
    try {
      const ms = await recordings.findCueTime(path, spans[0]!)
      if (!isOpen() || ms === null) return
      playback.go(ms)
    } catch (thrown) {
      if (!isOpen()) return
      content.error.value = formatErrorMessage(thrown)
    }
  }

  const goToLine = (line: number) => {
    const span = lines.spans.value[line]
    if (span) playback.go(span.from)
  }

  return { reach, goToLine }
}

function createTranscriptStatus(
  lines: ReturnType<typeof useTranscriptLines>,
  content: ReturnType<typeof useTranscriptContent>,
  playback: ReturnType<typeof usePlayback>,
) {
  const current = computed(() =>
    playback.hasTime.value ? findCueAt(lines.spans.value, playback.now.value) : -1,
  )
  const note = computed(() => {
    if (lines.times.value.length) return ''
    if (content.isWorking.value) return WORDS.transcribing
    return playback.points.value ? WORDS.unfetched : WORDS.silence
  })
  return { current, note }
}

export function useTranscript(recordings: Recordings, path: string, how: TranscriptOptions = {}) {
  const through = how.through ?? player
  const quiet = how.quiet ?? WRITE_IDLE_MS
  const plays = how.plays ?? createMediaTypeProbe()

  const following = ref(true)
  let open = true
  const isOpen = () => open

  const playback = usePlayback(through, plays, isOpen)
  const draftRef: { current?: ReturnType<typeof useTranscriptDraft> } = {}

  const content = useTranscriptContent(recordings, path, {
    setSource: playback.setSource,
    // Stryker disable next-line OptionalChaining: Assigned synchronously before any callback invocation.
    setText: (body) => draftRef.current?.setText(body),
    // Stryker disable next-line OptionalChaining: Assigned synchronously before any callback invocation.
    drop: () => draftRef.current?.drop(),
  })

  const draft = useTranscriptDraft({
    cues: content.cues,
    isEditable: content.isEditable,
    error: content.error,
    quiet,
    isOpen,
    write: (next) => recordings.writeTranscript(path, next),
  })
  draftRef.current = draft

  const lines = useTranscriptLines(content.cues, draft.prose)
  const { current, note } = createTranscriptStatus(lines, content, playback)
  const { reach, goToLine } = createTranscriptNavigation({
    recordings,
    path,
    content,
    playback,
    lines,
    isOpen,
  })

  return {
    path,
    ...playback,
    ...draft,
    ...lines,
    ...content,
    following,
    current,
    note,
    goToLine,
    setFollowing: (on: boolean) => {
      following.value = on
    },
    setWorking: (next: boolean) => content.setWorking(next, open),
    reach,
    close: () => {
      void draft.keep()
      playback.pause()
      open = false
      content.close()
      draft.stop()
    },
  }
}

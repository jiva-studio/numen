/**
 * One recording as its tab reads it: where the player stands in it, its transcript, which cue is
 * being said now, the words as a person edits them, and what its menu offers.
 */
import { computed, ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { findCueAt, getText, type Cue } from '../lib/cues'
import { useTranscriptDraft } from './draft'
import { useTranscriptLines } from './lines'
import { usePlayback } from './playback'
import { createMediaTypeProbe, player, type MediaTypeProbe, type Player } from './player'
import type { MenuEntry, Recordings, Span } from '../types'
import { WORDS } from '../words'

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

/** The entries of a menu a recording tab offers, each where the menu holds one. */
export function findOffers(entries: readonly MenuEntry[]) {
  return {
    transcribe: entries.find((one) => one.action === 'recognise'),
    proofread: entries.find((one) => one.action === 'proofread'),
    deleteText: entries.find((one) => one.action === 'deleteVersion' && one.isLive),
  }
}

/** Answers to a question asked again: only the latest asking may land. */
function createAnswerGuard() {
  let latest = 0
  let isClosed = false
  return {
    ask: () => {
      const mine = ++latest
      return { isLatest: () => !isClosed && mine === latest }
    },
    close: () => void (isClosed = true),
  }
}

export function useTranscript(recordings: Recordings, path: string, how: TranscriptOptions = {}) {
  const through = how.through ?? player
  const quiet = how.quiet ?? WRITE_IDLE_MS
  const plays = how.plays ?? createMediaTypeProbe()

  let isOpen = true
  const checkOpen = () => isOpen

  const cues = shallowRef<readonly Cue[]>([])
  const entries = shallowRef<readonly MenuEntry[]>([])
  const isWorking = ref(false)
  const isLoading = ref(true)
  const error = ref('')
  const following = ref(true)
  const isEditable = computed(() => !isWorking.value)
  const answers = createAnswerGuard()

  const playback = usePlayback(through, plays, checkOpen)
  const draft = useTranscriptDraft({
    cues,
    isEditable,
    error,
    quiet,
    isOpen: checkOpen,
    write: (next) => recordings.writeTranscript(path, next),
  })
  const lines = useTranscriptLines(cues, draft.prose)

  const readRecording = async () => {
    const mine = answers.ask()
    try {
      const [source, heard, menu] = await Promise.all([
        recordings.getSource(path),
        recordings.readTranscript(path),
        recordings.getMenu(path),
      ])
      if (!mine.isLatest()) return
      playback.setSource(source)
      cues.value = heard
      entries.value = menu
      draft.setText(getText(heard))
      error.value = ''
    } catch (thrown) {
      if (!mine.isLatest()) return
      error.value = formatErrorMessage(thrown)
    } finally {
      if (mine.isLatest()) isLoading.value = false
    }
  }

  const opened = readRecording()

  /** Work on the recording began or ended: the words and the menu are read again. */
  const setWorking = (next: boolean) => {
    if (!isOpen || (!next && !isWorking.value)) return
    isWorking.value = next
    void readRecording()
  }
  const stopWatching = recordings.watchWork(path, setWorking)

  const offers = computed(() => findOffers(entries.value))
  const isSettled = computed(() => !isLoading.value && !isWorking.value)
  const isTranscribable = computed(
    () => isSettled.value && !lines.timed.value && offers.value.transcribe !== undefined,
  )
  const isProofreadable = computed(
    () => isSettled.value && lines.timed.value && offers.value.proofread !== undefined,
  )
  const isDeletable = computed(
    () => isSettled.value && lines.timed.value && offers.value.deleteText !== undefined,
  )

  const runEntry = async (entry: MenuEntry | undefined) => {
    if (!entry || !isOpen) return
    try {
      await recordings.runMenuEntry(path, entry)
      if (entry.action === 'deleteVersion') {
        draft.drop()
        await readRecording()
      }
    } catch (thrown) {
      if (isOpen) error.value = formatErrorMessage(thrown)
    }
  }

  const current = computed(() => findCueAt(lines.spans.value, playback.now.value))
  const note = computed(() => {
    if (lines.times.value.length) return ''
    return isWorking.value ? WORDS.transcribing : WORDS.silence
  })

  /** The person is taken to the moment the first of the spans was said at. */
  const reach = async (...spans: readonly Span[]) => {
    await opened
    if (!isOpen || spans.length === 0) return
    try {
      const ms = await recordings.findCueTime(path, spans[0]!)
      if (isOpen && ms !== null) playback.controls.go(ms)
    } catch (thrown) {
      if (isOpen) error.value = formatErrorMessage(thrown)
    }
  }

  const goToLine = (line: number) => {
    const span = lines.spans.value[line]
    if (span) playback.controls.go(span.from)
  }

  const close = () => {
    void draft.keep()
    playback.controls.pause()
    isOpen = false
    stopWatching()
    answers.close()
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
      isWorking,
      isLoading,
      isEditable,
      error,
      opened,
      note,
      current,
      following,
      setFollowing: (isOn: boolean) => void (following.value = isOn),
      setWorking,
      reload: () => {
        draft.drop()
        void readRecording()
      },
    },
    offers: {
      isTranscribable,
      isProofreadable,
      isDeletable,
      transcribe: () => runEntry(offers.value.transcribe),
      proofread: () => runEntry(offers.value.proofread),
      deleteTranscript: () => runEntry(offers.value.deleteText),
    },
    navigation: { reach, goToLine },
    close,
  }
}

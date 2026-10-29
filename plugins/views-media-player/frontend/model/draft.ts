/**
 * The transcript text as the user edits it, and the debounced write that saves
 * it to the recording after typing stops.
 */
import { ref, type Ref } from 'vue'
import { applyCues, isSame, type Cue } from '../lib/cues'
import { formatErrorMessage } from '@numen/plugin-sdk'

/** What the words being edited are kept against. */
export interface DraftOptions {
  /** The words heard in the recording, in the order they were spoken. */
  cues: Ref<readonly Cue[]>
  /** Whether the transcript may be written over now. */
  isEditable: Ref<boolean>
  /** The last write error, shown above the transcript. */
  error: Ref<string>
  /** How long the typing settles for before the words are written. */
  idleMs: number
  /** Whether the recording tab is still open. */
  isOpen: () => boolean
  /** The transcript text as the user has edited it, saved to the recording. */
  write: (cues: readonly Cue[]) => Promise<void>
}

async function saveDraftCues(
  body: string,
  cues: Ref<readonly Cue[]>,
  options: { isOpen: () => boolean; write: (cues: readonly Cue[]) => Promise<void> },
): Promise<{ isSaved: boolean; errorText: string }> {
  const next = applyCues(cues.value, body)
  if (isSame(next, cues.value)) return { isSaved: true, errorText: '' }
  try {
    await options.write(next)
    if (!options.isOpen()) return { isSaved: false, errorText: '' }
    cues.value = next
    return { isSaved: true, errorText: '' }
  } catch (err) {
    if (!options.isOpen()) return { isSaved: false, errorText: '' }
    return { isSaved: false, errorText: formatErrorMessage(err) }
  }
}

export function useTranscriptDraft(how: DraftOptions) {
  const { cues, isEditable, error, idleMs, isOpen, write } = how

  /** The words as the editor shows them, one cue to a line. */
  const prose = ref('')
  /** Whether the user typed recently enough that the view must not follow playback. */
  const typing = ref(false)

  /** Whether what is on screen has still to reach the file. */
  let isPending = false
  /** A write of the words that has not answered yet. */
  let isWriting = false
  /** The debounce before keep() writes the edited words back. */
  let writeTimer: ReturnType<typeof setTimeout> | undefined
  /** The wait after which the view may go after the words again. */
  let followTimer: ReturnType<typeof setTimeout> | undefined

  /** Whether the words on screen have somewhere to go now. */
  const canWrite = (): boolean => isOpen() && isPending && !isWriting && isEditable.value

  /**
   * The words as they now read, kept against the recording. A write that is
   * refused leaves them isPending, so the next stillness offers them again.
   */
  const keep = async () => {
    clearTimeout(writeTimer)
    writeTimer = undefined
    if (!canWrite()) return
    const body = prose.value
    isPending = false
    isWriting = true
    try {
      const result = await saveDraftCues(body, cues, { isOpen, write })
      if (!isOpen()) return
      if (result.isSaved) {
        error.value = ''
        // Stryker disable next-line ConditionalExpression: An unsuccessful save carries errorText when open.
      } else if (result.errorText) {
        isPending = true
        error.value = result.errorText
      }
    } finally {
      isWriting = false
      if (isOpen() && prose.value !== body) {
        isPending = true
        writeTimer = setTimeout(() => void keep(), idleMs)
      }
    }
  }

  /** Applies the user's edit. The text is saved after typing stops. */
  const setProse = (body: string) => {
    if (!isOpen() || body === prose.value) return
    prose.value = body
    isPending = true
    // An edit that adds or removes a line shifts which line is active, and the
    // view does not follow a line the user moved while typing.
    typing.value = true
    clearTimeout(followTimer)
    followTimer = setTimeout(() => void (typing.value = false), idleMs)
    clearTimeout(writeTimer)
    writeTimer = setTimeout(() => void keep(), idleMs)
  }

  /**
   * Shows the recording's current text. Text the user typed and has not yet
   * saved is kept.
   */
  const setText = (body: string) => {
    if (!isPending) prose.value = body
  }

  /**
   * An edit still waiting to be written goes: it was of words that are no
   * longer the ones the recording has.
   */
  const drop = () => {
    clearTimeout(writeTimer)
    writeTimer = undefined
    isPending = false
  }

  /** Nothing more is typed here, and the words leave the screen. */
  const stop = () => {
    clearTimeout(followTimer)
    clearTimeout(writeTimer)
    writeTimer = undefined
    prose.value = ''
  }

  return { prose, typing, keep, setProse, setText, drop, stop }
}

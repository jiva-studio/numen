/**
 * The words as a person edits them, and the write that keeps them against the
 * recording once the typing has been still.
 */
import { ref, type Ref } from 'vue'
import { applyCues, same, type Cue } from '../lib/cues'
import { formatErrorMessage } from '@numen/wire'

/** What the words being edited are kept against. */
export interface DraftOptions {
  /** The words heard in the recording, in the order they were spoken. */
  cues: Ref<readonly Cue[]>
  /** Whether the transcript may be written over now. */
  isEditable: Ref<boolean>
  /** What this recording could not do, in words the tab puts up for it. */
  error: Ref<string>
  /** How long the typing settles for before the words are written. */
  quiet: number
  /** Whether the tab this recording stands in is still open. */
  isOpen: () => boolean
  /** The words as a person has edited them, kept against the recording. */
  write: (cues: readonly Cue[]) => Promise<void>
}

export function useTranscriptDraft(how: DraftOptions) {
  const { cues, isEditable, error, quiet, isOpen, write } = how

  /** The words as the editor shows them, one cue to a line. */
  const prose = ref('')
  /** Whether a person has been typing too recently for the view to move. */
  const typing = ref(false)

  /** Whether what is on screen has still to reach the file. */
  let owed = false
  /** A write of the words that has not answered yet. */
  let writing = false
  /** The wait the typing is being let settle over. */
  let settling: ReturnType<typeof setTimeout> | undefined
  /** The wait after which the view may go after the words again. */
  let stilling: ReturnType<typeof setTimeout> | undefined

  /** Whether the words on screen have somewhere to go now. */
  const canWrite = (): boolean => isOpen() && owed && !writing && isEditable.value

  /**
   * The words as they now read, kept against the recording. A write that is
   * refused leaves them owed, so the next stillness offers them again.
   */
  const keep = async () => {
    clearTimeout(settling)
    settling = undefined
    if (!canWrite()) return
    const body = prose.value
    const next = applyCues(cues.value, body)
    owed = false
    // A transcript written down is a transcript a person owns, and a
    // proofreader leaves it alone. Only words that changed are written.
    if (same(next, cues.value)) return
    writing = true
    try {
      await write(next)
      if (!isOpen()) return
      cues.value = next
      error.value = ''
    } catch (thrown) {
      if (!isOpen()) return
      owed = true
      error.value = formatErrorMessage(thrown)
    } finally {
      writing = false
      // Typing that landed while the write was in the air is still owed.
      if (isOpen() && prose.value !== body) {
        owed = true
        settling = setTimeout(() => void keep(), quiet)
      }
    }
  }

  /** The person typed. The words are written once they have been still. */
  const setProse = (body: string) => {
    if (!isOpen() || body === prose.value) return
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
   * The words the recording now has, put on screen. Words the person has typed
   * and not yet had written stay where they are.
   */
  const setText = (body: string) => {
    if (!owed) prose.value = body
  }

  /**
   * An edit still waiting to be written goes: it was of words that are no
   * longer the ones the recording has.
   */
  const drop = () => {
    clearTimeout(settling)
    settling = undefined
    owed = false
  }

  /** Nothing more is typed here, and the words leave the screen. */
  const stop = () => {
    clearTimeout(stilling)
    prose.value = ''
  }

  return { prose, typing, keep, setProse, setText, drop, stop }
}

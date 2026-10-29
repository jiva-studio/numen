/**
 * Pure state definitions, events, and effects for note tab editing and auto-save.
 */

/** Why writing or reading is impossible. */
export type NoteErrorCode =
  /** The file, or the body handed over, is past the ceiling. */
  | 'tooLarge'
  /** The core would not take the body it was given. */
  | 'bodyUnwritable'
  /** The vault does not hold this path as a note. */
  | 'notANote'
  /** The file is not valid UTF-8. */
  | 'notText'
  /** The frontmatter is broken, and the note can be neither read nor written. */
  | 'unreadable'
  /** The vault could not be reached. */
  | 'unreachable'

/**
 * The errors a keystroke is worth trying again after.
 */
export const RETRYABLE_ERRORS: readonly NoteErrorCode[] = [
  'tooLarge',
  'bodyUnwritable',
  'unreachable',
]

/** A note that moved to a new path. */
export interface Move {
  readonly from: string
  readonly to: string
}

/** What a tab last saw of its note: the prose a read gave it, and which file that read came out of. */
export interface NoteBaseline {
  readonly prose: string
  readonly stat: string
}

/** What a tab holds. */
export interface Tab {
  /** What was opened. */
  readonly path: string
  /** The body as last read or last successfully written, normalised. */
  readonly savedText: string | null
  /** Which file `written` came out of, and nothing where no file was read. */
  readonly filePath: string | null
  /** The body on screen. */
  readonly editorText: string
  /** The body being written to the file, or null when no write is pending. */
  readonly pendingWrite: string | null
  /** Whether the pending write has to be followed by another write. */
  readonly hasPendingWrite: boolean
  /** Whether the note this tab reads is at a name with no file behind it. */
  readonly isDeleted: boolean
  /** Whether the file moved past the prose this tab read, and so its save stopped. */
  readonly isStale: boolean
  /** When the first unwritten change was made, or nothing. */
  readonly since: number | null
  /** The generation of the newest read issued. */
  readonly reading: number
  /** Why writing or reading is impossible, or nothing. */
  readonly error: NoteErrorCode | null
}

export type State = 'loading' | 'stuck' | 'gone' | 'stale' | 'saving' | 'unsaved' | 'clean'

/** Dirty is what is shown differing from what was written. */
export const isDirty = (tab: Tab): boolean => tab.editorText !== tab.savedText

/**
 * The predicates are read in order, so a tab is in exactly one state.
 */
export const getState = (tab: Tab): State => {
  if (tab.error) return 'stuck'
  if (tab.savedText === null) return 'loading'
  if (tab.isDeleted) return 'gone'
  if (tab.isStale) return 'stale'
  if (tab.pendingWrite !== null) return 'saving'
  if (isDirty(tab)) return 'unsaved'
  return 'clean'
}

/**
 * The mark a tab carries beside its title.
 */
const MARKS: Record<State, string | undefined> = {
  stuck: 'stuck',
  gone: 'gone',
  stale: 'stale',
  unsaved: 'unsaved',
  saving: 'unsaved',
  loading: undefined,
  clean: undefined,
}

export const getMarkOf = (state: State): string | undefined => MARKS[state]

/** What a read answers. */
export type ReadResult =
  | { readonly kind: 'body'; readonly body: string; readonly at: string }
  | { readonly kind: 'missing' }
  | { readonly kind: 'error'; readonly error: NoteErrorCode }

/** What a write answers. */
export type WriteResult =
  | { readonly kind: 'ok'; readonly at: string }
  | { readonly kind: 'changed' }
  | { readonly kind: 'error'; readonly error: NoteErrorCode }

/** An event that a tab handles. */
export type Event =
  | { readonly kind: 'read'; readonly generation: number; readonly answer: ReadResult }
  | { readonly kind: 'edited'; readonly body: string; readonly at: number }
  | { readonly kind: 'timerFired' }
  | { readonly kind: 'written'; readonly answer: WriteResult }
  | {
      readonly kind: 'changed'
      readonly paths: readonly string[]
      readonly renamed: readonly Move[]
    }
  | { readonly kind: 'save' }
  | { readonly kind: 'flush' }
  | { readonly kind: 'keepLocal' }
  | { readonly kind: 'takeFile' }
  | { readonly kind: 'close' }

/** What should be done. */
export type Effect =
  | { readonly kind: 'read'; readonly path: string; readonly generation: number }
  | {
      readonly kind: 'write'
      readonly path: string
      readonly body: string
      readonly baseline: NoteBaseline | null
    }
  | { readonly kind: 'startTimer'; readonly after: number }
  | { readonly kind: 'cancelTimer' }
  | { readonly kind: 'replace'; readonly body: string }
  | { readonly kind: 'keepOpen' }
  | { readonly kind: 'reportError'; readonly error: NoteErrorCode }
  | { readonly kind: 'close' }

export interface Transition {
  readonly tab: Tab
  readonly effects: readonly Effect[]
}

/** The two limits a write waits on, in milliseconds. */
export interface WriteLimits {
  readonly quiet: number
  readonly bound: number
}

export const DEFAULT_WRITE_LIMITS: WriteLimits = { quiet: 800, bound: 5000 }

/** A tab as it opens: nothing read yet, and the first read issued. */
export const openTab = (path: string): Transition => ({
  tab: {
    path,
    savedText: null,
    filePath: null,
    editorText: '',
    pendingWrite: null,
    hasPendingWrite: false,
    isDeleted: false,
    isStale: false,
    since: null,
    reading: 1,
    error: null,
  },
  effects: [{ kind: 'read', path, generation: 1 }],
})

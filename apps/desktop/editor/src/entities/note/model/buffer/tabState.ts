/**
 * Pure state definitions, events, and effects for note tab editing and auto-save.
 */

/** Why writing or reading is impossible. */
export type NoteErrorCode =
  /** The file, or the body passed, is past the ceiling. */
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
  readonly path: string
  readonly savedText: string | null
  readonly revision: string | null
  readonly editorText: string
  readonly pendingWrite: string | null
  readonly hasPendingWrite: boolean
  readonly isDeleted: boolean
  readonly isStale: boolean
  readonly since: number | null
  readonly reading: number
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
    revision: null,
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

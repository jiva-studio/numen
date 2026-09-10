/**
 * Pure state machine transitions and effects for note tab editing and auto-save.
 */

/** Why writing or reading is impossible. */
export type Refusal =
  /** The file, or the body handed over, is past the ceiling. */
  | 'tooLarge'
  /** The core would not take the body it was given. */
  | 'bodyRefused'
  /** The vault does not hold this path as a note. */
  | 'notANote'
  /** The file is not valid UTF-8. */
  | 'notText'
  /** The frontmatter is broken, and the note can be neither read nor written. */
  | 'unreadable'
  /** The vault could not be reached. What is on screen is still here. */
  | 'unreachable'

/**
 * The refusals a keystroke is worth trying again after: the two a body causes,
 * and a vault that was out of reach when it was last asked.
 */
const mendable: readonly Refusal[] = ['tooLarge', 'bodyRefused', 'unreachable']

/** A note that is no longer where it was, and where it now is. */
export interface Move {
  readonly from: string
  readonly to: string
}

/** Which file prose came out of, as the core hands it back. */
export type FilePath = string

/** What a tab last saw of its note: the prose a read gave it, and which file that read came out of. */
export interface NoteBaseline {
  readonly prose: string
  readonly path: FilePath
}

/** What a tab holds. */
export interface Tab {
  /** What was opened. */
  readonly path: string
  /** The body as last read or last successfully written, normalised. */
  readonly written: string | null
  /** Which file `written` came out of, and nothing where no file was read. */
  readonly filePath: FilePath | null
  /** The body on screen. */
  readonly shown: string
  /** The body on its way to the file, and nothing where no write is in the air. */
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
  readonly error?: Refusal | null
  readonly refused: Refusal | null
}

export type State = 'loading' | 'stuck' | 'gone' | 'stale' | 'saving' | 'unsaved' | 'clean' | 'overtaken'

/** Dirty is what is shown differing from what was written. */
export const dirty = (tab: Tab): boolean => tab.shown !== tab.written

/**
 * The predicates are read in order, so a tab is in exactly one state. A read
 * that refused leaves no `written` behind it, and such a tab is stuck.
 */
export const stateOf = (tab: Tab): State => {
  if (tab.refused) return 'stuck'
  if (tab.written === null) return 'loading'
  if (tab.isDeleted) return 'gone'
  if (tab.isStale) return 'stale'
  if (tab.pendingWrite !== null) return 'saving'
  if (dirty(tab)) return 'unsaved'
  return 'clean'
}

/**
 * The one word a tab carries beside its title, and what a screen reader reads
 * out. A state with no word carries no mark; `stateOf` holds a tab in one
 * state, so the precedence the words are declared in is the precedence drawn.
 */
const MARKS: Record<State, string | undefined> = {
  stuck: 'stuck',
  gone: 'gone',
  stale: 'stale',
  overtaken: 'stale',
  unsaved: 'unsaved',
  saving: 'unsaved',
  loading: undefined,
  clean: undefined,
}

export const markOf = (state: State): string | undefined => MARKS[state]

/** What a read answers. */
export type ReadResult =
  | { readonly kind: 'body'; readonly body: string; readonly at: FilePath }
  | { readonly kind: 'missing' }
  | { readonly kind: 'refused'; readonly refusal: Refusal }

/** What a write answers. */
export type WriteResult =
  | { readonly kind: 'ok'; readonly at: FilePath }
  /** The file carries another fingerprint, and nothing was written. */
  | { readonly kind: 'changed' }
  | { readonly kind: 'refused'; readonly refusal: Refusal }

/** What a tab is told about. */
export type Event =
  /** A read of a generation answered. */
  | { readonly kind: 'read'; readonly generation: number; readonly answer: ReadResult }
  /** The person typed, and the whole body they left. */
  | { readonly kind: 'typed'; readonly body: string; readonly at: number }
  /** The interval fired. */
  | { readonly kind: 'fired' }
  /** The write in the air answered. */
  | { readonly kind: 'written'; readonly answer: WriteResult }
  /** The vault changed. A change carrying no paths is a reload. */
  | {
      readonly kind: 'changed'
      readonly paths: readonly string[]
      /** The notes that moved, so a tab showing one follows it. */
      readonly renamed: readonly Move[]
    }
  /** A save is asked for now. */
  | { readonly kind: 'saving' }
  /** The file is about to be renamed or removed. */
  | { readonly kind: 'settling' }
  /** The person keeps theirs: what is shown goes to the file. */
  | { readonly kind: 'keeping' }
  /** The person takes the file's: it is read again. */
  | { readonly kind: 'taking' }
  /** A close is asked for. */
  | { readonly kind: 'closing' }

/** What should be done. */
export type Effect =
  /** Issue a read of this generation. */
  | { readonly kind: 'read'; readonly path: string; readonly generation: number }
  /**
   * Begin a write, presenting what the tab last saw of the note. Nothing seen
   * writes over whatever the file holds.
   */
  | {
      readonly kind: 'write'
      readonly path: string
      readonly body: string
      readonly seen: NoteBaseline | null
    }
  /** Arm the interval to fire after this many milliseconds. */
  | { readonly kind: 'arm'; readonly after: number }
  /** Let go of the interval the tab is waiting on. */
  | { readonly kind: 'disarm' }
  /** Replace the document with this body. */
  | { readonly kind: 'replace'; readonly body: string }
  /** Keep the tab until the write in the air answers. */
  | { readonly kind: 'hold' }
  /** Say a refusal to the person. */
  | { readonly kind: 'say'; readonly refusal: Refusal }
  /** The tab goes. */
  | { readonly kind: 'close' }

export interface Transition {
  readonly tab: Tab
  readonly effects: readonly Effect[]
}

/** The two limits a write waits on, in milliseconds. */
export interface WriteLimits {
  /** How long the text has to have been still. */
  readonly quiet: number
  /** How old the first unwritten change gets before a write happens anyway. */
  readonly bound: number
}

export const waiting: WriteLimits = { quiet: 800, bound: 5000 }

/** A tab as it opens: nothing read yet, and the first read issued. */
export const opening = (path: string): Transition => ({
  tab: {
    path,
    written: null,
    filePath: null,
    shown: '',
    pendingWrite: null,
    hasPendingWrite: false,
    isDeleted: false,
    isStale: false,
    since: null,
    reading: 1,
    refused: null,
  },
  effects: [{ kind: 'read', path, generation: 1 }],
})


export const tabAfter = (tab: Tab, event: Event, limits: WriteLimits = waiting): Transition => {
  switch (event.kind) {
    case 'read':
      return answered(tab, event.generation, event.answer)
    case 'typed':
      return typed(tab, event.body, event.at, limits)
    case 'fired':
      return fired(tab)
    case 'written':
      return landed(tab, event.answer)
    case 'changed':
      return changed(tab, event.paths, event.renamed)
    case 'saving':
      return saving(tab)
    case 'settling':
      return settling(tab)
    case 'keeping':
      return keeping(tab)
    case 'taking':
      return taking(tab)
    case 'closing':
      return closing(tab)
  }
}

const still = (tab: Tab): Transition => ({ tab, effects: [] })

/**
 * A body equal to what is shown replaces nothing. What the file holds is what
 * the tab has written, so nothing is unwritten behind it.
 */
const shows = (tab: Tab, body: string, filePath: FilePath | null): Transition => ({
  tab: { ...tab, written: body, filePath, shown: body, since: null },
  effects: body === tab.shown ? [] : [{ kind: 'replace', body }],
})

/** What the tab last saw, for a write to present. */
const seenOf = (tab: Tab): NoteBaseline | null =>
  tab.written === null || tab.filePath === null ? null : { prose: tab.written, path: tab.filePath }

/** A write of what is on screen now, presenting what it is given. */
const begins = (tab: Tab, seen: NoteBaseline | null): Transition => ({
  tab: { ...tab, pendingWrite: tab.shown, hasPendingWrite: false, isStale: false },
  effects: [{ kind: 'write', path: tab.path, body: tab.shown, seen }],
})


/**
 * A read is applied to a tab with nothing unsaved and only for the generation
 * asked for last. An overtaken tab takes the body over what it shows, which is
 * the take it asked for.
 */
const answered = (tab: Tab, generation: number, answer: ReadResult): Transition => {
  const state = stateOf(tab)
  const loading = state === 'loading'
  // A read the tab has since replaced answers about the file it stood on then,
  // and says nothing about the one it stands on now.
  const stale = generation !== tab.reading
  if (answer.kind === 'refused') {
    return loading && !stale ? { tab: { ...tab, refused: answer.refusal }, effects: [] } : still(tab)
  }
  // Missing out of the first read empties the buffer, and the next write makes
  // the file. Missing out of a re-read is a note that is no longer there: what
  // the person has stays on the screen, and nothing of it is written anywhere
  // until they say so.
  if (answer.kind === 'missing') {
    if (stale) return still(tab)
    return loading ? shows(tab, '', null) : still({ ...tab, isDeleted: true })
  }
  if (loading) return shows(tab, answer.body, answer.at)
  // The note came back, at this name or another.
  if (state === 'gone') return shows({ ...tab, isDeleted: false }, answer.body, answer.at)
  if (stale) return still(tab)
  if (state === 'stale' || (state as string) === 'overtaken') return shows({ ...tab, isStale: false }, answer.body, answer.at)
  if (dirty(tab)) return still(tab)
  return shows(tab, answer.body, answer.at)
}

/**
 * When the write happens: once the text has been still for the quiet interval,
 * and no later than the bound after the first unwritten change.
 */
const armFor = (since: number, at: number, limits: WriteLimits): number =>
  Math.max(0, Math.min(limits.quiet, since + limits.bound - at))

/**
 * A refusal about the body is cleared by typing, and only where there is a
 * `written` for the typing to be dirty against. The others are conditions of the
 * file, which a keystroke does not mend.
 */
const mends = (tab: Tab): boolean =>
  tab.written !== null && tab.refused !== null && mendable.includes(tab.refused)

const typed = (tab: Tab, body: string, at: number, limits: WriteLimits): Transition => {
  const state = stateOf(tab)
  // A tab that has not read its note has no document to type into.
  if (state === 'loading') return still(tab)
  const refused = mends(tab) ? null : tab.refused
  const since = tab.since ?? at
  const next: Tab = { ...tab, shown: body, since, refused }
  // A stale tab arms nothing, and one of the two answers is what writes.
  if (state === 'stale' || (state as string) === 'overtaken') return still(next)
  return { tab: next, effects: [{ kind: 'arm', after: armFor(since, at, limits) }] }
}

const fired = (tab: Tab): Transition => {
  const state = stateOf(tab)
  // A tab that has not read its note has nothing to write.
  if (state === 'loading') return still(tab)
  if (state === 'unsaved') return begins(tab, seenOf(tab))
  if (state === 'saving') return { tab: { ...tab, hasPendingWrite: true }, effects: [] }
  return still(tab)
}

const landed = (tab: Tab, answer: WriteResult): Transition => {
  if (tab.pendingWrite === null) return still(tab)
  if (answer.kind === 'refused') {
    // The buffer stays editable, and what is owed goes.
    return {
      tab: { ...tab, pendingWrite: null, hasPendingWrite: false, refused: answer.refusal },
      effects: [],
    }
  }
  if (answer.kind === 'changed') {
    // Nothing was written, and the tab is stale until the person answers.
    return {
      tab: { ...tab, pendingWrite: null, hasPendingWrite: false, isStale: true, since: null },
      effects: [],
    }
  }
  const written: Tab = {
    ...tab,
    written: tab.pendingWrite,
    filePath: answer.at,
    pendingWrite: null,
    since: null,
    isDeleted: false,
  }
  return tab.hasPendingWrite ? begins(written, seenOf(written)) : still(written)
}

const changed = (tab: Tab, paths: readonly string[], renamed: readonly Move[]): Transition => {
  // A note that moved is followed wherever it went: its name changed and what
  // it holds did not. A tab left at the name it had holds a name with no file.
  const went = renamed.find((one) => one.from === tab.path)
  const next = went ? { ...tab, path: went.to, isDeleted: false } : tab

  // A change carrying no paths is a reload, and it is about every tab.
  const mine = paths.length === 0 || paths.includes(next.path) || went !== undefined
  if (!mine || stateOf(next) !== 'clean') return still(next)
  const reading = next.reading + 1
  return {
    tab: { ...next, reading },
    effects: [{ kind: 'read', path: next.path, generation: reading }],
  }
}

/**
 * A save asked for now writes what is unsaved and owes one to a write in the
 * air. Loading, stuck and overtaken write nothing.
 */
const saving = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'unsaved') return begins(tab, seenOf(tab))
  if (state === 'saving') return { tab: { ...tab, hasPendingWrite: true }, effects: [] }
  return still(tab)
}

/**
 * The file is about to be renamed or removed: the interval goes, and what is
 * unsaved reaches the path the tab still stands at. The tab stays where it is
 * and follows the note wherever it went.
 */
const settling = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state === 'unsaved') {
    const going = begins(tab, seenOf(tab))
    return { tab: going.tab, effects: [{ kind: 'disarm' }, ...going.effects] }
  }
  if (state === 'saving') {
    return { tab: { ...tab, hasPendingWrite: dirty(tab) }, effects: [{ kind: 'disarm' }] }
  }
  return { tab, effects: [{ kind: 'disarm' }] }
}

/** Keep: what is on screen goes to the file, whatever the file now holds. */
const keeping = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state !== 'stale' && (state as string) !== 'overtaken' && state !== 'gone') return still(tab)
  return begins(tab, null)
}

/** Take: the file is read again, and that read replaces the buffer. */
const taking = (tab: Tab): Transition => {
  const state = stateOf(tab)
  if (state !== 'stale' && (state as string) !== 'overtaken') return still(tab)
  const reading = tab.reading + 1
  return {
    tab: { ...tab, reading },
    effects: [{ kind: 'read', path: tab.path, generation: reading }],
  }
}

/**
 * A tab with something unwritten is held until what is owed answers, and the
 * caller closes it then. A stale tab is held with its question standing:
 * what the file holds and what the person typed are both still there, and
 * choosing between them is theirs. A tab that cannot be written says so, and
 * the buffer goes with it.
 */
const closing = (tab: Tab): Transition => {
  const state = stateOf(tab)
  // A refusal a keystroke or another try can mend holds the tab: the text is
  // still writable and the person has not been told it is about to go.
  if (tab.refused && state === 'stuck') {
    if (tab.written !== null && dirty(tab) && mendable.includes(tab.refused)) {
      return { tab, effects: [{ kind: 'say', refusal: tab.refused }, { kind: 'hold' }] }
    }
    return { tab, effects: [{ kind: 'say', refusal: tab.refused }, { kind: 'close' }] }
  }
  // A tab that has not read its note owes nothing.
  if (state === 'loading') return { tab, effects: [{ kind: 'close' }] }
  if (state === 'stale' || (state as string) === 'overtaken') return { tab, effects: [{ kind: 'hold' }] }
  if (state === 'unsaved') {
    const going = begins(tab, seenOf(tab))
    return { tab: going.tab, effects: [...going.effects, { kind: 'hold' }] }
  }
  if (state === 'saving') {
    return { tab: { ...tab, hasPendingWrite: dirty(tab) }, effects: [{ kind: 'hold' }] }
  }
  return { tab, effects: [{ kind: 'close' }] }
}

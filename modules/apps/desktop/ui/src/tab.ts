/**
 * What a tab holds, and what each event makes of it.
 *
 * A tab is the text on screen, the file it came from, and the write on its way
 * there. The rules that keep the three in step are here, so that a test can ask
 * them without a browser.
 *
 * Nothing happens here: an event is answered with the next tab and what should
 * be done — a read to issue, a write to begin, the interval to arm — and the
 * caller does it. The interval is a port, and the model is told that it fired.
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

/** The refusals that are about the body, which is what a keystroke can change. */
const aboutTheBody: readonly Refusal[] = ['tooLarge', 'bodyRefused']

/** The write on its way to the file. */
export interface Flight {
  readonly body: string
}

/** What a tab holds. */
export interface Tab {
  /** What was opened. */
  readonly path: string
  /** The body as last read or last successfully written, normalised. */
  readonly written: string | null
  /** The body on screen. */
  readonly shown: string
  readonly flight: Flight | null
  /** Whether the flight has to be followed by a write. */
  readonly owed: boolean
  /** When the first unwritten change was made, or nothing. */
  readonly since: number | null
  /** The generation of the newest read issued. */
  readonly reading: number
  /** Why writing or reading is impossible, or nothing. */
  readonly refused: Refusal | null
}

export type State = 'loading' | 'stuck' | 'saving' | 'unsaved' | 'clean'

/** Dirty is what is shown differing from what was written. */
export const dirty = (tab: Tab): boolean => tab.shown !== tab.written

/**
 * The predicates are read in order, so a tab is in exactly one state. A read
 * that refused leaves no `written` behind it, and such a tab is stuck.
 */
export const stateOf = (tab: Tab): State => {
  if (tab.refused) return 'stuck'
  if (tab.written === null) return 'loading'
  if (tab.flight) return 'saving'
  if (dirty(tab)) return 'unsaved'
  return 'clean'
}

/** What a read answers. */
export type Read =
  | { readonly kind: 'body'; readonly body: string }
  | { readonly kind: 'missing' }
  | { readonly kind: 'refused'; readonly refusal: Refusal }

/** What a write answers. */
export type Written =
  | { readonly kind: 'ok' }
  | { readonly kind: 'refused'; readonly refusal: Refusal }

/** What a tab is told about. */
export type Event =
  /** A read of a generation answered. */
  | { readonly kind: 'read'; readonly generation: number; readonly answer: Read }
  /** The person typed, and the whole body they left. */
  | { readonly kind: 'typed'; readonly body: string; readonly at: number }
  /** The interval fired. */
  | { readonly kind: 'fired' }
  /** The write in the air answered. */
  | { readonly kind: 'written'; readonly answer: Written }
  /** The vault changed. A change carrying no paths is a reload. */
  | { readonly kind: 'changed'; readonly paths: readonly string[] }
  /** A close is asked for. */
  | { readonly kind: 'closing' }

/** What should be done. */
export type Effect =
  /** Issue a read of this generation. */
  | { readonly kind: 'read'; readonly path: string; readonly generation: number }
  /** Begin a write. */
  | { readonly kind: 'write'; readonly path: string; readonly body: string }
  /** Arm the interval to fire after this many milliseconds. */
  | { readonly kind: 'arm'; readonly after: number }
  /** Replace the document with this body. */
  | { readonly kind: 'replace'; readonly body: string }
  /** Keep the tab until the write in the air answers. */
  | { readonly kind: 'hold' }
  /** Say a refusal to the person. */
  | { readonly kind: 'say'; readonly refusal: Refusal }
  /** The tab goes. */
  | { readonly kind: 'close' }

export interface Next {
  readonly tab: Tab
  readonly effects: readonly Effect[]
}

/** The two limits a write waits on, in milliseconds. */
export interface Waiting {
  /** How long the text has to have been still. */
  readonly quiet: number
  /** How old the first unwritten change gets before a write happens anyway. */
  readonly bound: number
}

export const waiting: Waiting = { quiet: 800, bound: 5000 }

/** A tab as it opens: nothing read yet, and the first read issued. */
export const opening = (path: string): Next => ({
  tab: {
    path,
    written: null,
    shown: '',
    flight: null,
    owed: false,
    since: null,
    reading: 1,
    refused: null,
  },
  effects: [{ kind: 'read', path, generation: 1 }],
})

export const tabAfter = (tab: Tab, event: Event, limits: Waiting = waiting): Next => {
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
      return changed(tab, event.paths)
    case 'closing':
      return closing(tab)
  }
}

const still = (tab: Tab): Next => ({ tab, effects: [] })

/** A body equal to what is shown replaces nothing. */
const shows = (tab: Tab, body: string): Next => ({
  tab: { ...tab, written: body, shown: body },
  effects: body === tab.shown ? [] : [{ kind: 'replace', body }],
})

/** A write of what is on screen now. */
const begins = (tab: Tab): Next => ({
  tab: { ...tab, flight: { body: tab.shown }, owed: false },
  effects: [{ kind: 'write', path: tab.path, body: tab.shown }],
})

/**
 * A read is applied to a tab with nothing unsaved and only for the generation
 * asked for last. Anything else is discarded, and the buffer is left alone.
 */
const answered = (tab: Tab, generation: number, answer: Read): Next => {
  const loading = stateOf(tab) === 'loading'
  if (answer.kind === 'refused') {
    return loading ? { tab: { ...tab, refused: answer.refusal }, effects: [] } : still(tab)
  }
  // Missing out of the first read empties the buffer, and the next write creates
  // the file. Missing out of a re-read leaves the buffer where it is.
  if (answer.kind === 'missing') return loading ? shows(tab, '') : still(tab)
  if (loading) return shows(tab, answer.body)
  if (dirty(tab) || generation !== tab.reading) return still(tab)
  return shows(tab, answer.body)
}

/**
 * When the write happens: once the text has been still for the quiet interval,
 * and no later than the bound after the first unwritten change.
 */
const armFor = (since: number, at: number, limits: Waiting): number =>
  Math.max(0, Math.min(limits.quiet, since + limits.bound - at))

/**
 * A refusal about the body is cleared by typing, and only where there is a
 * `written` for the typing to be dirty against. The others are conditions of the
 * file, which a keystroke does not mend.
 */
const mends = (tab: Tab): boolean =>
  tab.written !== null && tab.refused !== null && aboutTheBody.includes(tab.refused)

const typed = (tab: Tab, body: string, at: number, limits: Waiting): Next => {
  // A tab that has not read its note has no document to type into.
  if (stateOf(tab) === 'loading') return still(tab)
  const refused = mends(tab) ? null : tab.refused
  const since = tab.since ?? at
  return {
    tab: { ...tab, shown: body, since, refused },
    effects: [{ kind: 'arm', after: armFor(since, at, limits) }],
  }
}

const fired = (tab: Tab): Next => {
  const state = stateOf(tab)
  // A tab that has not read its note has nothing to write.
  if (state === 'loading') return still(tab)
  if (state === 'unsaved') return begins(tab)
  if (state === 'saving') return { tab: { ...tab, owed: true }, effects: [] }
  return still(tab)
}

const landed = (tab: Tab, answer: Written): Next => {
  if (!tab.flight) return still(tab)
  if (answer.kind === 'refused') {
    // The buffer stays editable, and what is owed goes.
    return { tab: { ...tab, flight: null, owed: false, refused: answer.refusal }, effects: [] }
  }
  const written: Tab = { ...tab, written: tab.flight.body, flight: null, since: null }
  return tab.owed ? begins(written) : still(written)
}

const changed = (tab: Tab, paths: readonly string[]): Next => {
  // A change carrying no paths is a reload, and it is about every tab.
  const mine = paths.length === 0 || paths.includes(tab.path)
  if (!mine || stateOf(tab) !== 'clean') return still(tab)
  const reading = tab.reading + 1
  return {
    tab: { ...tab, reading },
    effects: [{ kind: 'read', path: tab.path, generation: reading }],
  }
}

/**
 * A tab with something unwritten is held until what is owed answers, and the
 * caller closes it then. A tab that cannot be written says so, and the buffer
 * goes with it.
 */
const closing = (tab: Tab): Next => {
  const state = stateOf(tab)
  if (tab.refused && state === 'stuck') {
    return { tab, effects: [{ kind: 'say', refusal: tab.refused }, { kind: 'close' }] }
  }
  // A tab that has not read its note owes nothing.
  if (state === 'loading') return { tab, effects: [{ kind: 'close' }] }
  if (state === 'unsaved') {
    const going = begins(tab)
    return { tab: going.tab, effects: [...going.effects, { kind: 'hold' }] }
  }
  if (state === 'saving') return { tab: { ...tab, owed: dirty(tab) }, effects: [{ kind: 'hold' }] }
  return { tab, effects: [{ kind: 'close' }] }
}

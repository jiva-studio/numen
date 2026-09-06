/**
 * A change something other than the reader is making to the text.
 *
 * The text arrives in the document by the ordinary route and this draws over
 * it: the stretch about to be replaced is marked, and once the text is there
 * it is shown a word at a time from its start. Nothing here writes the text,
 * so a change that never lands leaves a dropped decoration and nothing else.
 */
import { Facet, StateEffect, StateField, type EditorState, type Range } from '@codemirror/state'
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
} from '@codemirror/view'
import { browserClock, type Clock } from '../plex/transition'

/** A change being made to this text by something other than the reader. */
export interface EditorChange {
  /** Whatever the caller addresses this change by. Nothing here reads it. */
  readonly id: string
  /** The stretch about to be replaced, as offsets into the text. */
  readonly from: number
  readonly to: number
  /** What is going in where that stretch stands. */
  readonly text: string
}

/** How much of a change's text is shown, and which of it arrived just now. */
export interface Reveal {
  /** Characters from the start of the text that are shown. */
  readonly shown: number
  /** Where the words shown for the first time begin; `shown` when none are. */
  readonly fading: number
}

/**
 * How long one word is shown arriving for, in milliseconds. It is what the
 * fade takes, so a word has finished arriving as the next one starts.
 */
const PACE = 220

/** The end of each word, taking in the space after it. */
const boundaries = (text: string): number[] => {
  const words = /\S+\s*/g
  const ends: number[] = []
  for (let found = words.exec(text); found; found = words.exec(text)) {
    ends.push(found.index + found[0].length)
  }
  return ends
}

/**
 * How much of `text` is shown `part` of the way through showing it.
 *
 * A word has a step to itself, and one step past the last word is where all of
 * the text stands and none of it is arriving.
 */
export const revealOf = (text: string, part: number): Reveal => {
  const ends = boundaries(text)
  const steps = ends.length + 1
  const at = Math.max(0, Math.min(steps, Math.floor(part * steps)))
  if (at === 0) return { shown: 0, fading: 0 }
  if (at === steps) return { shown: text.length, fading: text.length }
  return { shown: ends[at - 1] ?? text.length, fading: at > 1 ? (ends[at - 2] ?? 0) : 0 }
}

/** How long the whole of `text` takes to be shown. */
const timeOf = (text: string) => (boundaries(text).length + 1) * PACE

/** The change the editor is drawing over, and nothing when there is none. */
export const changing = Facet.define<EditorChange | null, EditorChange | null>({
  combine: (all) => all[0] ?? null,
})

/** How far the showing of the text has got. */
export const stepped = StateEffect.define<Reveal>()

const waiting = Decoration.mark({ class: 'cm-changing' })
const arriving = Decoration.mark({ class: 'cm-arriving' })
const covered = Decoration.replace({})

/** What is drawn for a change, and where it stands. */
interface Mark {
  readonly change: EditorChange | null
  readonly from: number
  readonly to: number
  /** How much of the text is shown; nothing until the text is there. */
  readonly reveal: Reveal | null
  readonly decorations: DecorationSet
}

const NOTHING: Mark = {
  change: null,
  from: 0,
  to: 0,
  reveal: null,
  decorations: Decoration.none,
}

const drawn = (
  change: EditorChange,
  from: number,
  to: number,
  reveal: Reveal | null,
): DecorationSet => {
  if (!reveal) return to > from ? Decoration.set([waiting.range(from, to)]) : Decoration.none

  const found: Range<Decoration>[] = []
  if (reveal.shown > reveal.fading)
    found.push(arriving.range(from + reveal.fading, from + reveal.shown))

  const end = from + change.text.length
  if (end > from + reveal.shown) found.push(covered.range(from + reveal.shown, end))
  return Decoration.set(found, true)
}

/**
 * Whether the text is in the document where the change said it would be. A
 * change that puts nothing in has no text to wait for.
 */
const arrived = (state: EditorState, change: EditorChange, from: number) =>
  change.text.length > 0 &&
  state.doc.sliceString(from, from + change.text.length) === change.text

const start = (state: EditorState): Mark => {
  const change = state.facet(changing)
  if (!change) return NOTHING

  const from = Math.max(0, Math.min(change.from, state.doc.length))
  const reveal = arrived(state, change, from) ? revealOf(change.text, 0) : null
  // Once the text is there, the stretch the change is about is the text.
  const to = reveal
    ? from + change.text.length
    : Math.max(from, Math.min(change.to, state.doc.length))
  return { change, from, to, reveal, decorations: drawn(change, from, to, reveal) }
}

/**
 * What is drawn over the change.
 *
 * Every range maps through the document's own changes, so typing beside a
 * change moves the overlay with the text under it.
 */
export const marked = StateField.define<Mark>({
  create: start,
  update: (was, transaction) => {
    if (transaction.state.facet(changing) !== transaction.startState.facet(changing))
      return start(transaction.state)

    const change = was.change
    if (!change) return was

    let stepping: Reveal | null = null
    for (const effect of transaction.effects) if (effect.is(stepped)) stepping = effect.value
    if (!stepping && !transaction.docChanged) return was

    const from = transaction.changes.mapPos(was.from, -1)
    const reveal =
      stepping ??
      (arrived(transaction.state, change, from) ? (was.reveal ?? revealOf(change.text, 0)) : null)
    const to = reveal
      ? from + change.text.length
      : Math.max(from, transaction.changes.mapPos(was.to, 1))

    return { change, from, to, reveal, decorations: drawn(change, from, to, reveal) }
  },
  provide: (field) => EditorView.decorations.from(field, (value) => value.decorations),
})

/** The clock that moves the boundary the words not yet shown stand behind. */
class Pace {
  private handle: number | null = null
  private started: number | null = null

  constructor(
    private readonly view: EditorView,
    private readonly clock: Clock,
  ) {
    this.take()
  }

  update(update: ViewUpdate) {
    const was = update.startState.field(marked)
    const now = update.state.field(marked)
    if (now.change !== was.change || (now.reveal === null) !== (was.reveal === null)) this.take()
  }

  destroy() {
    this.stop()
  }

  private take() {
    this.stop()
    const at = this.view.state.field(marked)
    if (!at.change || !at.reveal) return
    this.handle = this.clock.schedule(this.step)
  }

  private stop() {
    if (this.handle !== null) this.clock.cancel(this.handle)
    this.handle = null
    this.started = null
  }

  // Elapsed time comes from the timestamp each callback is handed.
  private readonly step = (now: number) => {
    this.handle = null
    const at = this.view.state.field(marked)
    if (!at.change || !at.reveal) return

    this.started ??= now
    const ms = timeOf(at.change.text)
    const part = ms <= 0 ? 1 : Math.min(1, (now - this.started) / ms)
    const reveal = revealOf(at.change.text, part)
    if (reveal.shown !== at.reveal.shown || reveal.fading !== at.reveal.fading)
      this.view.dispatch({ effects: stepped.of(reveal) })
    if (part < 1) this.handle = this.clock.schedule(this.step)
  }
}

export const pacing = (clock: Clock = browserClock) =>
  ViewPlugin.define((view) => new Pace(view, clock))

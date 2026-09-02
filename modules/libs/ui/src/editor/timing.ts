/**
 * A time against every line, and the line being read now.
 *
 * The times stand in the gutter and are clicked to go to one. Which line is
 * being read, and whether the view moves to keep it in sight, are the caller's
 * to say.
 */
import { StateEffect, StateField, type Extension } from '@codemirror/state'
import { Decoration, EditorView, GutterMarker, ViewPlugin, gutter } from '@codemirror/view'

/** What the editor shows against its lines. */
export interface Timed {
  /** What stands in the gutter, one for each line from the first. */
  readonly times: readonly string[]
  /** The line being read, counted from zero. None is -1. */
  readonly now: number
  /** The view moves to keep the line being read in sight. */
  readonly follows: boolean
}

const NOTHING: Timed = { times: [], now: -1, follows: false }

const told = StateEffect.define<Timed>()

const held = StateField.define<Timed>({
  create: () => NOTHING,
  update: (was, transaction) => {
    for (const effect of transaction.effects) if (effect.is(told)) return effect.value
    return was
  },
})

/** The line being read. It is drawn in the accent and carries no fill. */
const reading = Decoration.line({ class: 'cm-reading' })

const marked = EditorView.decorations.compute([held, 'doc'], (state) => {
  const { now } = state.field(held)
  if (now < 0 || now >= state.doc.lines) return Decoration.none
  return Decoration.set([reading.range(state.doc.line(now + 1).from)])
})

/** One time in the gutter. It is clicked to go to the line it stands on. */
class Time extends GutterMarker {
  override elementClass: string

  constructor(
    readonly text: string,
    readonly at: number,
    readonly now: boolean,
    readonly goes: (line: number) => void,
  ) {
    super()
    this.elementClass = now ? 'cm-reading' : ''
  }

  override eq(other: Time): boolean {
    return other.text === this.text && other.at === this.at && other.now === this.now
  }

  override toDOM(): Node {
    const mark = document.createElement('button')
    mark.type = 'button'
    mark.className = 'cm-time'
    // The gutter holds one of these for every line on screen, and the tab key
    // walks past all of them to the words.
    mark.tabIndex = -1
    mark.textContent = this.text
    mark.addEventListener('mousedown', (event) => {
      event.preventDefault()
      this.goes(this.at)
    })
    return mark
  }
}

const times = (goes: (line: number) => void): Extension =>
  gutter({
    class: 'cm-times',
    lineMarker: (view, line) => {
      const at = view.state.doc.lineAt(line.from).number - 1
      const { times: all, now } = view.state.field(held)
      const text = all[at]
      return text === undefined ? null : new Time(text, at, at === now, goes)
    },
    lineMarkerChange: (update) => update.startState.field(held) !== update.state.field(held),
  })

/* The times take the room the longest of them needs, so an hour in and a
   minute in line up. */
const painted = EditorView.theme({
  '.cm-gutters': {
    backgroundColor: 'transparent',
    border: 'none',
    color: 'inherit',
  },
  '.cm-times': {
    minWidth: '4ch',
    fontVariantNumeric: 'tabular-nums',
    fontSize: '0.85em',
  },
  '.cm-times .cm-gutterElement': {
    padding: '0 0.5rem 0 0',
    textAlign: 'end',
  },
  '.cm-time': {
    padding: '0',
    border: '0',
    background: 'none',
    color: 'var(--numen-hushed)',
    font: 'inherit',
    cursor: 'pointer',
    userSelect: 'none',
  },
  '.cm-time:focus-visible': { outline: 'none' },
  '.cm-reading, .cm-reading .cm-time': {
    color: 'var(--numen-focus-border)',
  },
})

/** The times of one editor, and what puts them there. */
export interface Timing {
  /** The extension, put in the editor these are shown in. */
  readonly extension: Extension
  /** What the editor shows now. */
  show(timed: Timed): void
}

/** Whether two of these say the same thing. */
const same = (one: Timed, two: Timed): boolean =>
  one.now === two.now &&
  one.follows === two.follows &&
  (one.times === two.times ||
    (one.times.length === two.times.length &&
      one.times.every((text, at) => text === two.times[at])))

export function timing(goes: (line: number) => void): Timing {
  let view: EditorView | null = null

  // What was last shown. An editor drawn again — a tab moved, a pane split —
  // is a new editor holding none of it, and is given it as it attaches.
  let last: Timed | null = null

  const holding = ViewPlugin.define((got) => {
    view = got
    if (last) {
      const shown = last
      queueMicrotask(() => {
        if (view === got) show(shown)
      })
    }
    return {
      destroy: () => {
        if (view === got) view = null
      },
    }
  })

  const show = (timed: Timed) => {
    last = timed
    if (!view) return
    const was = view.state.field(held)
    if (same(was, timed)) return
    const effects: StateEffect<unknown>[] = [told.of(timed)]
    // The view is moved only where following is on and the line to keep in
    // sight is one the document has.
    const moved = timed.now !== was.now || timed.follows !== was.follows
    if (timed.follows && moved && timed.now >= 0 && timed.now < view.state.doc.lines) {
      const { from } = view.state.doc.line(timed.now + 1)
      effects.push(EditorView.scrollIntoView(from, { y: 'nearest' }))
    }
    view.dispatch({ effects })
  }

  return { extension: [held, marked, times(goes), painted, holding], show }
}

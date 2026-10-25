/**
 * A time against every line, and the line being said now.
 *
 * The times stand in the gutter and are clicked to go to one. Which line is
 * being said, and whether the view moves to keep it in sight, are the caller's
 * to say.
 */
import { StateEffect, StateField, type Extension } from '@codemirror/state'
import { Decoration, EditorView, GutterMarker, ViewPlugin, gutter } from '@codemirror/view'

/** What the editor shows against its lines. */
export interface TimeGutterState {
  readonly times: readonly string[]
  /** The line being said, counted from zero. None is -1. */
  readonly current: number
  /** The view moves to keep the line being said in sight. */
  readonly isFollowing: boolean
}

const NOTHING: TimeGutterState = { times: [], current: -1, isFollowing: false }

const setTimeGutterState = StateEffect.define<TimeGutterState>()

const timeGutterField = StateField.define<TimeGutterState>({
  create: () => NOTHING,
  update: (was, transaction) => {
    for (const effect of transaction.effects) if (effect.is(setTimeGutterState)) return effect.value
    return was
  },
})

/** The line being said. It is drawn in the accent and carries no fill. */
const currentLine = Decoration.line({ class: 'cm-current' })

const currentLineMarks = EditorView.decorations.compute([timeGutterField, 'doc'], (state) => {
  const lineIndex = state.field(timeGutterField).current
  if (lineIndex < 0 || lineIndex >= state.doc.lines) return Decoration.none
  return Decoration.set([currentLine.range(state.doc.line(lineIndex + 1).from)])
})

interface TimeOptions {
  readonly text: string
  readonly at: number
  readonly isCurrent: boolean
  readonly goToLine: (line: number) => void
}

/** One time in the gutter. It is clicked to go to the line it relies on. */
class Time extends GutterMarker {
  override elementClass: string
  readonly text: string
  readonly at: number
  readonly isCurrent: boolean
  readonly goToLine: (line: number) => void

  constructor(options: TimeOptions) {
    super()
    this.text = options.text
    this.at = options.at
    this.isCurrent = options.isCurrent
    this.goToLine = options.goToLine
    this.elementClass = options.isCurrent ? 'cm-current' : ''
  }

  override eq(other: Time): boolean {
    return other.text === this.text && other.at === this.at && other.isCurrent === this.isCurrent
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
      this.goToLine(this.at)
    })
    return mark
  }
}

const createTimeGutterExtension = (goToLine: (line: number) => void): Extension =>
  gutter({
    class: 'cm-times',
    lineMarker: (view, line) => {
      const lineIndex = view.state.doc.lineAt(line.from).number - 1
      const { times: all, current } = view.state.field(timeGutterField)
      const text = all[lineIndex]
      return text === undefined
        ? null
        : new Time({ text, at: lineIndex, isCurrent: lineIndex === current, goToLine })
    },
    lineMarkerChange: (update) =>
      update.startState.field(timeGutterField) !== update.state.field(timeGutterField),
  })

/* The times take the room the longest of them needs, so an hour in and a
   minute in line up. */
const timeGutterTheme = EditorView.theme({
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
    display: 'flex',
    alignItems: 'flex-end',
    justifyContent: 'flex-end',
    padding: '0 0.5rem 0 0',
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
  '.cm-current, .cm-current .cm-time': {
    color: 'var(--numen-accent)',
  },
})

/** The times of one editor, and what puts them there. */
export interface TimeGutter {
  /** The extension, put in the editor these are shown in. */
  readonly extension: Extension
  /** What the editor shows now. */
  show(state: TimeGutterState): void
}

/** Whether two of these say the same thing. */
const isSame = (one: TimeGutterState, two: TimeGutterState): boolean =>
  one.current === two.current &&
  one.isFollowing === two.isFollowing &&
  one.times.length === two.times.length &&
  one.times.every((text, index) => text === two.times[index])

export function createTimeGutter(goToLine: (line: number) => void): TimeGutter {
  let view: EditorView | null = null

  // What was last shown. An editor drawn again — a tab moved, a pane split —
  // is a new editor attachPlugin none of it, and is given it as it attaches.
  let last: TimeGutterState | null = null

  const attachPlugin = ViewPlugin.define((got) => {
    view = got
    if (last) {
      const lastTime = last
      queueMicrotask(() => {
        // An editor drawn again is shown what the one before it was shown and
        // is left at its current position: nothing was said while it was drawn.
        if (view === got) applyState(lastTime, false)
      })
    }
    return {
      destroy: () => {
        if (view === got) view = null
      },
    }
  })

  // put shows what is given. It moves the view only where it may: the line to
  // keep in sight has to be one the document has, following has to be on, and
  // one of the two has to have changed.
  const applyState = (state: TimeGutterState, may: boolean) => {
    last = state
    if (!view) return
    const was = view.state.field(timeGutterField)
    if (isSame(was, state)) return
    const effects: StateEffect<unknown>[] = [setTimeGutterState.of(state)]
    const isMoved = state.current !== was.current || state.isFollowing !== was.isFollowing
    if (
      may &&
      state.isFollowing &&
      isMoved &&
      state.current >= 0 &&
      state.current < view.state.doc.lines
    ) {
      const { from } = view.state.doc.line(state.current + 1)
      effects.push(EditorView.scrollIntoView(from))
    }
    view.dispatch({ effects })
  }

  const show = (state: TimeGutterState) => applyState(state, true)

  return {
    extension: [
      timeGutterField,
      currentLineMarks,
      createTimeGutterExtension(goToLine),
      timeGutterTheme,
      attachPlugin,
    ],
    show,
  }
}

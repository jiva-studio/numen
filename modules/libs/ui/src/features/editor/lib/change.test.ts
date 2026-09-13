/**
 * How much of a change is shown, as plain values, and what is drawn for it.
 *
 * The arithmetic has no clock in it: a moment of the showing is a value the
 * tests name. The clock is a port, wound here by hand.
 */
import { describe, expect, it } from 'vitest'
import { EditorState, StateEffect } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { changing, marked, createPacePlugin, revealOf, stepped, type EditorChange } from './change'
import type { Clock } from '@/shared/lib/clock'
import { createState } from '../fixtures/state'
import { RUSSIAN } from '@/shared/fixtures/prose'

// Nothing here has a size, and the editor measures anyway.
Range.prototype.getClientRects = () =>
  Object.assign([], { item: () => null }) as unknown as DOMRectList
Range.prototype.getBoundingClientRect = () => new DOMRect()

/** Five words, whose ends are at 4, 8, 14, 19 and 23. */
const FIVE = 'one two three four five'

interface Drawn {
  readonly from: number
  readonly to: number
  /** What the stretch is marked as, and nothing where it is covered over. */
  readonly mark: string | null
}

const getDrawn = (state: EditorState): Drawn[] => {
  const found: Drawn[] = []
  state.field(marked).decorations.between(0, state.doc.length, (from, to, deco) => {
    found.push({ from, to, mark: (deco.spec as { class?: string }).class ?? null })
  })
  return found
}

/** A state with the parser the editor runs, and a change put in over it. */
const createMarkedState = (doc: string, change: EditorChange | null): EditorState =>
  createState(doc).update({
    effects: StateEffect.appendConfig.of([marked, changing.of(change)]),
  }).state

/** The showing, moved on to `part` of the way through. */
const getStateAt = (state: EditorState, change: EditorChange, part: number): EditorState =>
  state.update({ effects: stepped.of(revealOf(change.text, part)) }).state

describe('how much of a change is shown', () => {
  it('is none of it before it starts', () => {
    expect(revealOf(FIVE, 0)).toEqual({ shown: 0, fading: 0 })
  })

  it('is two words of five, forty per cent of the way through', () => {
    expect(revealOf(FIVE, 0.4)).toEqual({ shown: 8, fading: 4 })
  })

  it('is one more word at each step', () => {
    const steps = [0, 1, 2, 3, 4, 5, 6].map((step) => revealOf(FIVE, step / 6).shown)
    expect(steps).toEqual([0, 4, 8, 14, 19, 23, 23])
  })

  it('takes in the space after a word, so the boundary stands between words', () => {
    expect(revealOf(FIVE, 0.2).shown).toBe(4)
  })

  it('is all of it at the end, with nothing arriving', () => {
    expect(revealOf(FIVE, 1)).toEqual({ shown: 23, fading: 23 })
  })

  it('stays inside the text however far it is asked for', () => {
    expect(revealOf(FIVE, 4)).toEqual({ shown: 23, fading: 23 })
    expect(revealOf(FIVE, -1)).toEqual({ shown: 0, fading: 0 })
  })

  it('has nothing to show for text with no words in it', () => {
    expect(revealOf('', 0.5)).toEqual({ shown: 0, fading: 0 })
    expect(revealOf('   \n  ', 0.5)).toEqual({ shown: 0, fading: 0 })
  })

  it('counts the words of a script that is not Latin', () => {
    expect(revealOf(RUSSIAN, 1)).toEqual({ shown: RUSSIAN.length, fading: RUSSIAN.length })
    expect(revealOf(RUSSIAN, 0.5).shown).toBeGreaterThan(0)
    expect(revealOf(RUSSIAN, 0.5).shown).toBeLessThan(RUSSIAN.length)
  })
})

describe('a change on its way', () => {
  const DOC = 'the cat sat on the mat'
  const CHANGE: EditorChange = { id: 'a', from: 4, to: 7, text: 'dog' }

  it('marks the stretch it is about to replace', () => {
    expect(getDrawn(createMarkedState(DOC, CHANGE))).toEqual([{ from: 4, to: 7, mark: 'cm-changing' }])
  })

  it('is drawn for nobody while there is no change', () => {
    expect(getDrawn(createMarkedState(DOC, null))).toEqual([])
  })

  it('moves with the text when something is typed before it', () => {
    const typed = createMarkedState(DOC, CHANGE).update({ changes: { from: 0, insert: 'so ' } }).state
    expect(getDrawn(typed)).toEqual([{ from: 7, to: 10, mark: 'cm-changing' }])
  })

  it('is drawn for nothing where it is addressed past the end of the text', () => {
    const past: EditorChange = { id: 'a', from: 40, to: 90, text: 'dog' }
    expect(getDrawn(createMarkedState(DOC, past))).toEqual([])
  })

  it('leaves the text as it was', () => {
    expect(createMarkedState(DOC, CHANGE).doc.toString()).toBe(DOC)
  })
})

describe('a change that puts nothing in', () => {
  it('marks the stretch it takes out, and there is nothing to show', () => {
    const gone: EditorChange = { id: 'a', from: 4, to: 8, text: '' }
    expect(getDrawn(createMarkedState('the cat sat', gone))).toEqual([{ from: 4, to: 8, mark: 'cm-changing' }])
  })
})

describe('a change whose text has arrived', () => {
  const DOC = 'the dog and cat sat'
  const CHANGE: EditorChange = { id: 'a', from: 4, to: 7, text: 'dog and cat' }

  it('covers all of the text before any of it is shown', () => {
    expect(getDrawn(createMarkedState(DOC, CHANGE))).toEqual([{ from: 4, to: 15, mark: null }])
  })

  it('shows the words that have had their turn and covers the rest', () => {
    expect(getDrawn(getStateAt(createMarkedState(DOC, CHANGE), CHANGE, 0.4))).toEqual([
      { from: 4, to: 8, mark: 'cm-arriving' },
      { from: 8, to: 15, mark: null },
    ])
  })

  it('draws nothing at all once every word is shown', () => {
    expect(getDrawn(getStateAt(createMarkedState(DOC, CHANGE), CHANGE, 1))).toEqual([])
  })

  it('takes over from the mark when the text lands in the document', () => {
    const waiting = createMarkedState('the cat sat', { id: 'a', from: 4, to: 7, text: 'dog' })
    expect(getDrawn(waiting)).toEqual([{ from: 4, to: 7, mark: 'cm-changing' }])

    const landed = waiting.update({ changes: { from: 4, to: 7, insert: 'dog' } }).state
    expect(getDrawn(landed)).toEqual([{ from: 4, to: 7, mark: null }])
  })

  it('goes back to the mark when the text is typed away from under it', () => {
    const shown = getStateAt(createMarkedState(DOC, CHANGE), CHANGE, 0.4)
    const typed = shown.update({ changes: { from: 5, to: 6, insert: 'i' } }).state
    expect(getDrawn(typed)).toEqual([{ from: 4, to: 15, mark: 'cm-changing' }])
  })
})

/** A clock the test winds by hand. */
const createClock = () => {
  let pending: ((now: number) => void) | null = null
  const clock: Clock = {
    now: () => 0,
    schedule: (run) => {
      pending = run
      return 1
    },
    cancel: () => {
      pending = null
    },
  }
  return {
    clock,
    /** Hand the frame that is waiting its timestamp. */
    tick: (now: number) => {
      const run = pending
      pending = null
      run?.(now)
    },
  }
}

describe('the showing, stepped by the clock', () => {
  const DOC = 'the dog and cat sat'
  const CHANGE: EditorChange = { id: 'a', from: 4, to: 7, text: 'dog and cat' }

  const wound = () => {
    const world = createClock()
    const view = new EditorView({
      parent: document.body,
      state: EditorState.create({
        doc: DOC,
        extensions: [marked, changing.of(CHANGE), createPacePlugin(world.clock)],
      }),
    })
    return { world, view }
  }

  it('shows one word more at each turn of it', () => {
    const { world, view } = wound()

    world.tick(0)
    expect(getDrawn(view.state)).toEqual([{ from: 4, to: 15, mark: null }])

    world.tick(220)
    expect(getDrawn(view.state)).toEqual([
      { from: 4, to: 8, mark: 'cm-arriving' },
      { from: 8, to: 15, mark: null },
    ])

    world.tick(440)
    expect(getDrawn(view.state)).toEqual([
      { from: 8, to: 12, mark: 'cm-arriving' },
      { from: 12, to: 15, mark: null },
    ])

    world.tick(660)
    expect(getDrawn(view.state)).toEqual([{ from: 12, to: 15, mark: 'cm-arriving' }])

    world.tick(880)
    expect(getDrawn(view.state)).toEqual([])

    view.destroy()
  })

  it('stops asking for frames once all of it is shown', () => {
    const { world, view } = wound()
    for (const now of [0, 220, 440, 660, 880]) world.tick(now)

    const settled = getDrawn(view.state)
    world.tick(1100)
    expect(getDrawn(view.state)).toEqual(settled)

    view.destroy()
  })

  it('never writes the text', () => {
    const { world, view } = wound()
    for (const now of [0, 220, 440, 660, 880]) world.tick(now)
    expect(view.state.doc.toString()).toBe(DOC)

    view.destroy()
  })
})

/**
 * Where in a book the reader stands, and what turning a page evaluates to.
 *
 * The document is four spreads of one column, each holding one run. The style
 * of the drawn text is stood in for, so a test reads what was written to it and
 * in what order.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed, shallowRef } from 'vue'

import type { Flow, Mark } from '../lib/spread'
import type { SettledBookProps } from '../lib/props'
import { useBookTurns } from './turns'

const FLOW: Flow = { contentWidth: 4000, width: 1000, gap: 0, columns: 1 }
const MARKS: readonly Mark[] = [
  { at: 0, x: 0 },
  { at: 100, x: 1000 },
  { at: 200, x: 2000 },
  { at: 300, x: 3000 },
]

const paperWithStyle = () => {
  const writes: string[] = []
  const paper = document.createElement('div')
  const style = {
    set translate(value: string) {
      writes.push(`translate ${value}`)
    },
    set transition(value: string) {
      writes.push(`transition [${value}]`)
    },
  }
  Object.defineProperty(paper, 'style', { value: style })
  return { paper, writes }
}

const setUp = (
  marks: readonly Mark[] = MARKS,
  offset = 0, rendered = true) => {
  const { paper, writes } = paperWithStyle()
  const props = { book: { from: 0, to: 400 }, span: { from: 0, to: 400 }, at: offset } as SettledBookProps
  const onMove = vi.fn<(pos: number) => void>()
  const turns = useBookTurns({
    paper: shallowRef<HTMLElement | null>(rendered ? paper : null),
    props,
    flow: computed(() => FLOW),
    spreadCount: computed(() => 4),
    marks: shallowRef(marks),
    onMove,
  })
  return { turns, writes, onMove }
}

describe('how much of the chapter is still to come', () => {
  it('is the columns after the spread in front', () => {
    const { turns } = setUp()
    expect(turns.leftInChapter.value).toBe(3)

    turns.stand(1, 'auto')

    expect(turns.leftInChapter.value).toBe(2)
  })
})

describe('putting a spread in front', () => {
  it('keeps the spread within the document', () => {
    const { turns } = setUp()

    turns.stand(10, 'auto')
    expect(turns.standing.value).toBe(3)

    turns.stand(-2, 'auto')
    expect(turns.standing.value).toBe(0)
  })

  it('carries the columns to the spread at once, without a transition', () => {
    const { turns, writes } = setUp()

    turns.stand(2, 'auto')

    expect(writes).toStrictEqual(['transition [none]', 'translate -2000px 0', 'transition []'])
  })

  it('carries the columns with the transition where the turn is smooth', () => {
    const { turns, writes } = setUp()

    turns.stand(2, 'smooth')

    expect(writes).toStrictEqual(['translate -2000px 0'])
  })

  it('counts the spread without a drawn document', () => {
    const { turns, writes } = setUp(MARKS, 0, false)

    expect(() => turns.stand(2, 'auto')).not.toThrow()
    expect(turns.standing.value).toBe(2)
    expect(writes).toStrictEqual([])
  })
})

describe('going to a spread', () => {
  it('carries the columns with the transition and says what remains in front', () => {
    const { turns, writes, onMove } = setUp()

    turns.goTo(1)

    expect(writes).toStrictEqual(['translate -1000px 0'])
    expect(onMove).toHaveBeenCalledExactlyOnceWith(100)
  })

  it('says nothing where the offset in front is the one already retained', () => {
    const { turns, onMove } = setUp(MARKS, 100)

    turns.goTo(1)

    expect(onMove).not.toHaveBeenCalled()
  })

  it('says nothing where no run remains in the spread', () => {
    const { turns, onMove } = setUp(MARKS.slice(0, 2))

    turns.goTo(3)

    expect(onMove).not.toHaveBeenCalled()
  })
})

describe('the offset kept in front', () => {
  it('is that of the first run in the spread in front', () => {
    const { turns } = setUp(MARKS, 7)

    turns.stand(2, 'auto')

    expect(turns.getKeptOffset()).toBe(200)
  })

  it('is the offset retained where no run remains in the spread in front', () => {
    const { turns } = setUp(MARKS.slice(0, 2), 7)

    turns.stand(3, 'auto')

    expect(turns.getKeptOffset()).toBe(7)
  })
})

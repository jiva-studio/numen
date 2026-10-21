/** Where a card lands, and which runs draw a drop target of their own. */
import { describe, expect, it } from 'vitest'
import type { Stencil } from './card'
import { createRunEnd, HEAD, type DeckCard, type DeckSection } from './deck'
import { getGrid, isMoved } from './grid'

const CUTS: readonly Stencil[] = [{ name: 'Animal', fields: ['Name'] }]
const SECTIONS: readonly DeckSection[] = [
  { id: 'roots', name: 'Roots' },
  { id: 'leaves', name: 'Leaves' },
]
const card = (id: string, section: string | null): DeckCard => ({
  id,
  section,
  stencil: 'Animal',
  filled: [],
})

const grid = (
  cards: readonly DeckCard[],
  sections: readonly DeckSection[] = [],
  dragCard: string | null = null,
) => getGrid({ cards, sections, stencils: CUTS }, dragCard)

describe('getGrid, the drop target a run draws', () => {
  it('draws no drop target for the cards before the first section where there is no section', () => {
    expect(grid([], []).runs.map((run) => run.isDropTarget)).toEqual([false])
  })

  it('draws a drop target for the empty head of a deck that has sections', () => {
    expect(grid([], SECTIONS).runs[0]?.isDropTarget).toBe(true)
  })

  it('draws none for a head holding cards, the first of them being landed on', () => {
    expect(grid([card('a', null)], SECTIONS).runs[0]?.isDropTarget).toBe(false)
  })

  it('draws none for a run under a section, its heading being landed on', () => {
    const runs = grid([card('a', 'roots')], SECTIONS).runs
    expect(runs.slice(1).map((run) => run.isDropTarget)).toEqual([false, false])
  })

  it('remains a card naming a section that is not there before the first section', () => {
    const runs = grid([card('a', 'gone')], SECTIONS).runs
    expect(runs[0]?.tiles.map((tile) => [tile.id, tile.section])).toEqual([['a', null]])
  })
})

describe('isMoved', () => {
  const getRuns = (cards: readonly DeckCard[]) => grid(cards, SECTIONS).runs
  const RUNS = getRuns([card('a', null), card('b', null), card('c', 'roots')])

  it('moves nothing let go on itself', () => {
    expect(isMoved(RUNS, 'a', 'a')).toBe(false)
  })

  it('moves nothing let go at the head when it is the first card there', () => {
    expect(isMoved(RUNS, 'a', HEAD)).toBe(false)
    expect(isMoved(RUNS, 'b', HEAD)).toBe(true)
  })

  it('moves a card let go at the head of a deck with no card', () => {
    expect(isMoved([], 'a', HEAD)).toBe(true)
  })

  it('moves nothing let go at the end of the run it ends', () => {
    expect(isMoved(RUNS, 'c', createRunEnd('roots'))).toBe(false)
    expect(isMoved(RUNS, 'a', createRunEnd('roots'))).toBe(true)
  })

  it('moves a card let go at the end of a run holding none', () => {
    expect(isMoved(RUNS, 'a', createRunEnd('leaves'))).toBe(true)
  })

  it('moves a card let go at the end of a run that is not there', () => {
    expect(isMoved(RUNS, 'a', createRunEnd('gone'))).toBe(true)
  })

  it('moves a card let go before another card, or at the end of nothing', () => {
    expect(isMoved(RUNS, 'a', 'c')).toBe(true)
    expect(isMoved(RUNS, 'a', null)).toBe(true)
  })
})

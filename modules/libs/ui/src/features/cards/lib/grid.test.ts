/** Where a card lands, and which runs draw a landing of their own. */
import { describe, expect, it } from 'vitest'
import type { Stencil } from './card'
import { endOf, HEAD, type DeckCard, type DeckSection } from './deck'
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

describe('getGrid, the landing a run draws', () => {
  it('draws no landing for the cards before the first section where there is no section', () => {
    expect(getGrid([], [], CUTS, null).runs.map((run) => run.isLanding)).toEqual([false])
  })

  it('draws a landing for the empty head of a deck that has sections', () => {
    expect(getGrid([], SECTIONS, CUTS, null).runs[0]?.isLanding).toBe(true)
  })

  it('draws none for a head holding cards, the first of them being landed on', () => {
    expect(getGrid([card('a', null)], SECTIONS, CUTS, null).runs[0]?.isLanding).toBe(false)
  })

  it('draws none for a run under a section, its heading being landed on', () => {
    const runs = getGrid([card('a', 'roots')], SECTIONS, CUTS, null).runs
    expect(runs.slice(1).map((run) => run.isLanding)).toEqual([false, false])
  })

  it('stands a card naming a section that is not there before the first section', () => {
    const runs = getGrid([card('a', 'gone')], SECTIONS, CUTS, null).runs
    expect(runs[0]?.tiles.map((tile) => [tile.id, tile.section])).toEqual([['a', null]])
  })
})

describe('isMoved', () => {
  const runsOf = (cards: readonly DeckCard[]) => getGrid(cards, SECTIONS, CUTS, null).runs
  const RUNS = runsOf([card('a', null), card('b', null), card('c', 'roots')])

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
    expect(isMoved(RUNS, 'c', endOf('roots'))).toBe(false)
    expect(isMoved(RUNS, 'a', endOf('roots'))).toBe(true)
  })

  it('moves a card let go at the end of a run holding none', () => {
    expect(isMoved(RUNS, 'a', endOf('leaves'))).toBe(true)
  })

  it('moves a card let go at the end of a run that is not there', () => {
    expect(isMoved(RUNS, 'a', endOf('gone'))).toBe(true)
  })

  it('moves a card let go before another card, or at the end of nothing', () => {
    expect(isMoved(RUNS, 'a', 'c')).toBe(true)
    expect(isMoved(RUNS, 'a', null)).toBe(true)
  })
})

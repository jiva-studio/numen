import { describe, expect, it } from 'vitest'
import { placeTurns, VOICES, type Turn } from './model'

const said = (id: string, text = 'said', state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'asked', text } : { id, voice: 'asked', text, state }

const back = (id: string, text = 'back', state?: Turn['state']): Turn =>
  state === undefined
    ? { id, voice: 'answered', text }
    : { id, voice: 'answered', text, state }

describe('the voices', () => {
  it('draws what was said in a bubble and what came back on the surface', () => {
    expect(VOICES.asked.bubble).toBe(true)
    expect(VOICES.answered.bubble).toBe(false)
  })

  it('puts them on opposite sides', () => {
    expect(VOICES.asked.against).toBe('end')
    expect(VOICES.answered.against).toBe('start')
  })

})

describe('placing the turns', () => {
  it('has nothing to place in an empty thread', () => {
    expect(placeTurns([])).toStrictEqual([])
  })

  it('carries each turn through untouched', () => {
    const turns = [said('1'), back('2')]
    expect(placeTurns(turns).map((placed) => placed.turn)).toStrictEqual(turns)
  })

  it('settles a turn that does not say otherwise', () => {
    expect(placeTurns([said('1')])[0]?.state).toBe('settled')
  })

  it('keeps the state a turn came with', () => {
    expect(placeTurns([said('1', 'gone', 'failed')])[0]?.state).toBe('failed')
  })

  it('hands each turn the descriptor for its voice', () => {
    expect(placeTurns([said('1')])[0]?.voice).toBe(VOICES.asked)
    expect(placeTurns([back('1')])[0]?.voice).toBe(VOICES.answered)
  })

  it('puts the caret on an answer still arriving at the end', () => {
    const placed = placeTurns([said('1'), back('2', 'half a s', 'arriving')])
    expect(placed[1]?.caret).toBe(true)
  })

  it('withholds the caret from an arriving turn that something follows', () => {
    const placed = placeTurns([back('1', 'left behind', 'arriving'), said('2')])
    expect(placed[0]?.caret).toBe(false)
  })

  it('withholds the caret from a turn that settled', () => {
    expect(placeTurns([back('1')])[0]?.caret).toBe(false)
  })

  it('withholds the caret from a turn that failed', () => {
    expect(placeTurns([back('1', 'gone', 'failed')])[0]?.caret).toBe(false)
  })

  it('places a turn with no text at all', () => {
    const placed = placeTurns([said('1', '')])
    expect(placed).toHaveLength(1)
    expect(placed[0]?.turn.text).toBe('')
  })
})

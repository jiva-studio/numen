import { describe, expect, it } from 'vitest'
import { placeTurns, VOICES, type Turn } from './turn'

const createAsked = (id: string, text = 'said', state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'asked', text } : { id, voice: 'asked', text, state }

const createAnswered = (id: string, text = 'back', state?: Turn['state']): Turn =>
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
    const turns = [createAsked('1'), createAnswered('2')]
    expect(placeTurns(turns).map((placed) => placed.turn)).toStrictEqual(turns)
  })

  it('settles a turn that does not say otherwise', () => {
    expect(placeTurns([createAsked('1')])[0]?.state).toBe('settled')
  })

  it('keeps the state a turn came with', () => {
    expect(placeTurns([createAsked('1', 'gone', 'failed')])[0]?.state).toBe('failed')
  })

  it('hands each turn the descriptor for its voice', () => {
    expect(placeTurns([createAsked('1')])[0]?.voice).toBe(VOICES.asked)
    expect(placeTurns([createAnswered('1')])[0]?.voice).toBe(VOICES.answered)
  })

  it('places a turn with no text at all', () => {
    const placed = placeTurns([createAsked('1', '')])
    expect(placed).toHaveLength(1)
    expect(placed[0]?.turn.text).toBe('')
  })
})

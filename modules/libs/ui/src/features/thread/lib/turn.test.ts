import vm from 'node:vm'
import { setFlagsFromString } from 'node:v8'
import { describe, expect, it } from 'vitest'
import { placeTurns, VOICES, writeCharCount, type Turn } from './turn'

setFlagsFromString('--expose-gc')

const createAsked = (id: string, text = 'said', state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'asked', text } : { id, voice: 'asked', text, state }

const createAnswered = (id: string, text = 'back', state?: Turn['state']): Turn =>
  state === undefined ? { id, voice: 'answered', text } : { id, voice: 'answered', text, state }

describe('the voices', () => {
  it('draws the transcript in a bubble and what came back on the surface', () => {
    expect(VOICES.asked.isBubble).toBe(true)
    expect(VOICES.answered.isBubble).toBe(false)
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

  it('passes each turn the descriptor for its voice', () => {
    expect(placeTurns([createAsked('1')])[0]?.voice).toBe(VOICES.asked)
    expect(placeTurns([createAnswered('1')])[0]?.voice).toBe(VOICES.answered)
  })

  it('places a turn with no text at all', () => {
    const placedTurns = placeTurns([createAsked('1', '')])
    expect(placedTurns).toHaveLength(1)
    expect(placedTurns[0]?.turn.text).toBe('')
  })
})

describe('placing the same turns again', () => {
  it('returns the entry a turn was given before', () => {
    const turns = [createAsked('1'), createAnswered('2')]
    const first = placeTurns(turns)
    const second = placeTurns([...turns])
    expect(second[0]).toBe(first[0])
    expect(second[1]).toBe(first[1])
  })

  it('keeps the entries of earlier turns while the last one changes', () => {
    const earlier = [createAsked('1'), createAnswered('2')]
    const first = placeTurns([...earlier, createAnswered('3', 'th')])
    const second = placeTurns([...earlier, createAnswered('3', 'thi')])
    expect(second[0]).toBe(first[0])
    expect(second[1]).toBe(first[1])
    expect(second[2]).not.toBe(first[2])
    expect(second[2]?.turn.text).toBe('thi')
  })

  it('does not keep a turn that was removed', async () => {
    const collect = vm.runInNewContext('gc') as () => void
    let isCollected = false
    const registry = new FinalizationRegistry(() => (isCollected = true))
    const place = () => {
      const turn = createAnswered('1', 'removed')
      registry.register(turn, 'removed')
      placeTurns([turn])
    }
    place()

    for (let attempt = 0; attempt < 50 && !isCollected; attempt++) {
      await new Promise((resolve) => setTimeout(resolve, 10))
      collect()
    }

    expect(isCollected).toBe(true)
    expect(registry).toBeDefined()
  })
})

describe('writeCharCount', () => {
  it('says nothing for no characters', () => {
    expect(writeCharCount(0)).toBe('')
    expect(writeCharCount(-3)).toBe('')
  })

  it('counts characters', () => {
    expect(writeCharCount(1)).toBe('1 characters')
    expect(writeCharCount(999)).toBe('999 characters')
  })
})

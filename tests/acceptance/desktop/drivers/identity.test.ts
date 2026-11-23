import { describe, expect, it } from 'vitest'
import { getBuyName, getBuyPattern, getChangeName, getPlanLine, planAnswers, readPosition } from './identity.js'

const WORDS = ['zoo', 'zoo', 'cat', 'dog', 'zoo', 'fox', 'owl', 'hen', 'ant', 'bee', 'elk', 'yak']

describe('readPosition', () => {
  it('reads the place a confirmation asks for', () => {
    expect(readPosition('Confirm word 4')).toBe(4)
    expect(readPosition('Confirm word 12')).toBe(12)
  })

  it('refuses a label that is not a confirmation', () => {
    expect(() => readPosition('Recovery words')).toThrow('not the label of a confirmation')
  })
})

describe('planAnswers', () => {
  it('returns the word at each requested position', () => {
    expect(planAnswers([3, 4, 12], WORDS)).toStrictEqual(['cat', 'dog', 'yak'])
  })

  it('puts the mistake in the place of a word that is not the mistake', () => {
    expect(planAnswers([1, 3, 4], WORDS, 'zoo')).toStrictEqual(['zoo', 'zoo', 'dog'])
    expect(planAnswers([1, 2, 3], WORDS, 'zoo')).toStrictEqual(['zoo', 'zoo', 'zoo'])
  })

  it('refuses a place the words do not have', () => {
    expect(() => planAnswers([13], WORDS)).toThrow('word 13 of 12')
  })

  it('rejects a wrong word at every requested position', () => {
    expect(() => planAnswers([1, 2, 5], WORDS, 'zoo')).toThrow('nothing can take its place')
  })
})

describe('the names of the buttons of the card', () => {
  it('names the button that buys a plan with its name and size', () => {
    expect(getBuyName('Plus', 100)).toBe('Buy Plus, 100 GB')
    expect(getBuyPattern('Plus').test('Buy Plus, 100 GB')).toBe(true)
    expect(getBuyPattern('Plus').test('Buy Pro, 250 GB')).toBe(false)
  })

  it('names the button that changes the plan with the name of the plan', () => {
    expect(getChangeName('Pro')).toBe('Change plan to Pro')
  })

  it('writes the plan line with the name and the size', () => {
    expect(getPlanLine('Plus', 100)).toBe('Plus, 100 GB')
  })
})

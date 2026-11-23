import { describe, expect, it } from 'vitest'
import { formatDay, getChooseName, getPlanLinePattern, planAnswers, readPosition } from './identity.js'

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
  it('names the button that chooses a plan with the name of the plan', () => {
    expect(getChooseName('Plus')).toBe('Choose Plus')
  })

  it('writes a day with the month in full', () => {
    expect(formatDay(new Date(2026, 11, 7))).toBe('December 7, 2026')
  })
})

describe('getPlanLinePattern', () => {
  it('matches the plan row of an active and of an ended subscription', () => {
    expect(getPlanLinePattern('Plus', 100).test('100 GB, $15 / month')).toBe(true)
    expect(getPlanLinePattern('Plus', 100).test('Plus, 100 GB, $15 / month')).toBe(true)
    expect(getPlanLinePattern('Plus', 100).test('100 GB')).toBe(true)
  })

  it('does not match another size or the detail of a plan to choose', () => {
    expect(getPlanLinePattern('Plus', 100).test('10 GB, $5 / month')).toBe(false)
    expect(getPlanLinePattern('Plus', 100).test('100 GB of synced files')).toBe(false)
  })
})

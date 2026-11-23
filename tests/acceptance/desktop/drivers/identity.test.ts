import { describe, expect, it } from 'vitest'
import { formatDay, getChoosePattern, getPlanLinePattern, pickOtherWords, planAnswers, planSheetTexts, readPosition, readSheetNumbers } from './identity.js'

const WORDS = ['zoo', 'zoo', 'cat', 'dog', 'zoo', 'fox', 'owl', 'hen', 'ant', 'bee', 'elk', 'yak']

describe('readPosition', () => {
  it('reads the place a confirmation asks for', () => {
    expect(readPosition('Word 4')).toBe(4)
    expect(readPosition('Word 12')).toBe(12)
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
  it('names the row that chooses a plan with the name, the size and the price of the plan', () => {
    expect(getChoosePattern('Plus').test('Choose Plus, 100 GB, $15 / month')).toBe(true)
    expect(getChoosePattern('Plus', 100).test('Choose Plus, 100 GB, $15 / month')).toBe(true)
    expect(getChoosePattern('Plus', 10).test('Choose Plus, 100 GB, $15 / month')).toBe(false)
    expect(getChoosePattern('Plus').test('Choose Plus Pro, 100 GB')).toBe(false)
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

const SHEETS = [
  Array.from({ length: 33 }, (_, index) => `a${index + 1}`),
  Array.from({ length: 33 }, (_, index) => `b${index + 1}`),
  Array.from({ length: 33 }, (_, index) => `c${index + 1}`),
]

describe('readSheetNumbers', () => {
  it('reads one sheet and two sheets in the order given', () => {
    expect(readSheetNumbers('2')).toStrictEqual([2])
    expect(readSheetNumbers('1 and 3')).toStrictEqual([1, 3])
    expect(readSheetNumbers('3 and 1')).toStrictEqual([3, 1])
  })

  it('refuses a text that does not name sheets of a kit of three', () => {
    expect(() => readSheetNumbers('four')).toThrow('does not name sheets')
    expect(() => readSheetNumbers('0')).toThrow('does not name sheets')
    expect(() => readSheetNumbers('1 and 4')).toThrow('does not name sheets')
    expect(() => readSheetNumbers('')).toThrow('does not name sheets')
  })
})

describe('planSheetTexts', () => {
  it('writes the words of each named sheet with one space between them', () => {
    expect(planSheetTexts(SHEETS, [3, 1])).toStrictEqual([SHEETS[2]?.join(' '), SHEETS[0]?.join(' ')])
  })

  it('replaces exactly one word of the sheet with the mistake and no other sheet', () => {
    const [first, second] = planSheetTexts(SHEETS, [1, 2], 2)
    expect(first).toBe(SHEETS[0]?.join(' '))
    const changed = (second ?? '').split(' ')
    expect(changed).toHaveLength(33)
    const differences = changed.filter((word, at) => word !== SHEETS[1]?.[at])
    expect(differences).toHaveLength(1)
  })

  it('replaces the word with another word of the list that is not the word itself', () => {
    const sheets = [Array.from({ length: 33 }, () => 'academic'), ...SHEETS.slice(1)]
    const [text] = planSheetTexts(sheets, [1], 1)
    expect(text).not.toBe(sheets[0]?.join(' '))
    expect((text ?? '').split(' ')).toHaveLength(33)
  })

  it('refuses a mistake in a sheet that is not typed and a sheet the kit does not have', () => {
    expect(() => planSheetTexts(SHEETS, [1, 2], 3)).toThrow('not among the sheets')
    expect(() => planSheetTexts(SHEETS.slice(0, 1), [2])).toThrow('no sheet 2')
  })
})

describe('pickOtherWords', () => {
  it('gives twelve words that are not the words given', () => {
    const own = 'legal winner thank year wave sausage worth useful legal winner thank yellow'.split(' ')
    const other = pickOtherWords(own)
    expect(other).toHaveLength(12)
    expect(other.join(' ')).not.toBe(own.join(' '))
    expect(pickOtherWords(other).join(' ')).not.toBe(other.join(' '))
  })
})

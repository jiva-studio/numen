/**
 * What the field of a proof recognizes in the text typed or pasted into it. Every word is invented.
 */
import { describe, expect, it } from 'vitest'
import { readProof } from './readProof'

const createWords = (prefix: string, count: number): string[] =>
  Array.from({ length: count }, (_, index) => `${prefix}${index + 1}`)

const PHRASE = createWords('w', 12)
const SHEET_ONE = createWords('a', 33)
const SHEET_TWO = createWords('b', 33)
const SHEET_THREE = createWords('c', 33)

describe('readProof of the 12 words', () => {
  it('recognizes 12 words on one line', () => {
    const found = readProof(PHRASE.join(' '))
    expect(found.proof).toStrictEqual({ words: PHRASE })
    expect(found.note).toBe('12 words recognized')
    expect(found.isProblem).toBe(false)
  })

  it('recognizes 12 words one to a line and in capitals', () => {
    const found = readProof(PHRASE.map((word) => ` ${word.toUpperCase()} `).join('\n'))
    expect(found.proof).toStrictEqual({ words: PHRASE })
  })

  it('counts the words so far without a proof', () => {
    const found = readProof(PHRASE.slice(0, 5).join(' '))
    expect(found.proof).toBeUndefined()
    expect(found.note).toBe('5 of 12 words')
    expect(found.isProblem).toBe(false)
  })

  it('refuses a 13th word with the count', () => {
    const found = readProof(`${PHRASE.join(' ')} extra`)
    expect(found.proof).toBeUndefined()
    expect(found.note).toContain('13 words')
    expect(found.isProblem).toBe(true)
  })

  it('says nothing for an empty text or white space', () => {
    expect(readProof('')).toStrictEqual({ proof: undefined, note: '', isProblem: false })
    expect(readProof(' \n  ').note).toBe('')
  })
})

describe('readProof of the sheets of a kit', () => {
  it('recognizes two sheets, a sheet on each line', () => {
    const found = readProof(`${SHEET_ONE.join(' ')}\n${SHEET_THREE.join(' ')}`)
    expect(found.proof).toStrictEqual({ sheets: [SHEET_ONE, SHEET_THREE] })
    expect(found.note).toBe('2 of 3 sheets recognized')
    expect(found.isProblem).toBe(false)
  })

  it('needs one more sheet after the first, with no proof', () => {
    const found = readProof(SHEET_ONE.join(' '))
    expect(found.proof).toBeUndefined()
    expect(found.note).toBe('Sheet 1 recognized, one more sheet needed')
    expect(found.isProblem).toBe(false)
  })

  it('counts the words of a sheet that is not whole yet', () => {
    const found = readProof(`${SHEET_ONE.join(' ')}\n${SHEET_TWO.slice(0, 20).join(' ')}`)
    expect(found.proof).toBeUndefined()
    expect(found.note).toBe('Sheet 2 has 20 of 33 words')
    expect(found.isProblem).toBe(false)
  })

  it('refuses a sheet with a word too many and names the sheet', () => {
    const found = readProof(`${SHEET_ONE.join(' ')}\n${[...SHEET_TWO, 'extra'].join(' ')}`)
    expect(found.proof).toBeUndefined()
    expect(found.note).toContain('Sheet 2 has 34 words')
    expect(found.isProblem).toBe(true)
  })

  it('refuses a repeated sheet and names both', () => {
    const found = readProof(`${SHEET_ONE.join(' ')}\n${SHEET_ONE.join('  ').toUpperCase()}`)
    expect(found.proof).toBeUndefined()
    expect(found.note).toBe('Sheet 2 is the same as sheet 1. Enter a different sheet.')
    expect(found.isProblem).toBe(true)
  })

  it('points to the first problem when there are several', () => {
    const found = readProof(
      [SHEET_ONE, SHEET_ONE, SHEET_TWO].map((one) => one.join(' ')).join('\n'),
    )
    expect(found.note).toContain('Sheet 2 is the same as sheet 1')
  })

  it('refuses a third sheet', () => {
    const found = readProof(
      [SHEET_ONE, SHEET_TWO, SHEET_THREE].map((one) => one.join(' ')).join('\n'),
    )
    expect(found.proof).toBeUndefined()
    expect(found.note).toBe('Two sheets are enough. Remove the others.')
    expect(found.isProblem).toBe(true)
  })

  it('reads a sheet of one line that is neither 12 words nor a sheet as a wrong count', () => {
    const found = readProof(SHEET_ONE.slice(0, 20).join(' '))
    expect(found.proof).toBeUndefined()
    expect(found.note).toContain('20 words')
    expect(found.isProblem).toBe(true)
  })

  it('reads 33 words wrapped into short lines as too many words', () => {
    const lines = [0, 11, 22].map((from) => SHEET_ONE.slice(from, from + 11).join(' '))
    const found = readProof(lines.join('\n'))
    expect(found.proof).toBeUndefined()
    expect(found.note).toContain('33 words')
    expect(found.isProblem).toBe(true)
  })
})

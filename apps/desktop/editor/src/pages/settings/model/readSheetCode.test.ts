import { describe, expect, it } from 'vitest'
import { addScannedSheet, readSheetCode } from './readSheetCode'

const WORDS = Array.from({ length: 33 }, (_, index) => `word${String.fromCharCode(97 + (index % 26))}`)
const makeCode = (kit: string, sheet: number, list: readonly string[] = WORDS): string =>
  `numen://recover?v=1&kit=${kit}&sheet=${sheet}&words=${list.join('+')}`

describe('readSheetCode', () => {
  it('reads the kit, the sheet and the words of a sheet code', () => {
    expect(readSheetCode(makeCode('C0DE', 2))).toEqual({
      code: { kit: 'C0DE', sheet: 2, words: WORDS },
    })
  })

  it.each([
    ['text that is no link', 'hello'],
    ['another link', 'https://example.com/?v=1'],
    ['a newer version', makeCode('C0DE', 1).replace('v=1', 'v=2')],
    ['a sheet of 12 words', makeCode('C0DE', 1, WORDS.slice(0, 12))],
    ['a sheet number of 4', makeCode('C0DE', 4)],
  ])('refuses %s in plain words', (_name, text) => {
    expect(readSheetCode(text).problem).toBe('This is not a code of a Numen recovery kit.')
  })
})

describe('addScannedSheet', () => {
  it('takes a second sheet of the same kit after the first', () => {
    const first = addScannedSheet(undefined, makeCode('C0DE', 1))
    expect(first.next).toEqual({ kit: 'C0DE', sheets: [1] })
    expect(first.line).toBe(WORDS.join(' '))
    expect(addScannedSheet(first.next, makeCode('C0DE', 3)).next).toEqual({
      kit: 'C0DE',
      sheets: [1, 3],
    })
  })

  it('refuses a sheet scanned twice and says which', () => {
    const first = addScannedSheet(undefined, makeCode('C0DE', 2))
    expect(addScannedSheet(first.next, makeCode('C0DE', 2)).problem).toBe(
      'Sheet 2 is scanned already. Scan another sheet.',
    )
  })

  it('refuses a sheet of another kit', () => {
    const first = addScannedSheet(undefined, makeCode('C0DE', 1))
    expect(addScannedSheet(first.next, makeCode('BEEF', 2)).problem).toBe(
      'This sheet is from another kit. Scan a sheet of the same kit.',
    )
  })
})

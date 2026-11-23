import { describe, expect, it } from 'vitest'
import { findRepeatedParts, parsePartLog, type PartLogEntry } from './part-log.js'

const put = (blob: string, part: number): PartLogEntry => ({ blob, part, direction: 'put', from: 0, bytes: 5_242_880 })
const read = (blob: string, from: number, bytes: number): PartLogEntry => ({ blob, part: 0, direction: 'get-range', from, bytes })

describe('findRepeatedParts', () => {
  it('finds nothing in parts that each went up once, also of two blobs with the same numbers', () => {
    expect(findRepeatedParts([put('a', 1), put('a', 2), put('b', 1), put('b', 2)])).toStrictEqual([])
  })

  it('finds the part that went up twice', () => {
    expect(findRepeatedParts([put('a', 1), put('a', 2), put('a', 2)])).toStrictEqual([put('a', 2)])
  })

  it('finds nothing in ranges that follow one another', () => {
    expect(findRepeatedParts([read('a', 0, 1000), read('a', 1000, 500), read('a', 1500, 20)])).toStrictEqual([])
  })

  it('finds the range that reads bytes a range before it read', () => {
    expect(findRepeatedParts([read('a', 0, 1000), read('a', 900, 500)])).toStrictEqual([read('a', 900, 500)])
  })

  it('does not mix the ranges of two blobs or a range with a part put', () => {
    expect(findRepeatedParts([read('a', 0, 1000), read('b', 0, 1000), put('a', 1)])).toStrictEqual([])
  })
})

describe('parsePartLog', () => {
  it('reads the entries of the server', () => {
    const entries = [put('a', 1), read('a', 10, 20)]

    expect(parsePartLog(JSON.stringify(entries))).toStrictEqual(entries)
  })

  it('refuses text that is not a list of entries', () => {
    expect(() => parsePartLog('{}')).toThrow('part log')
    expect(() => parsePartLog('[{"blob":"a"}]')).toThrow('part log')
    expect(() => parsePartLog('nonsense')).toThrow('part log')
  })
})

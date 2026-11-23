import { describe, expect, it } from 'vitest'
import { appendToFixture, buildFixture } from './file-fixtures.js'

const PNG_SIGNATURE = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])

describe('buildFixture', () => {
  it('starts a ".pdf" file with the header of a PDF', () => {
    expect(buildFixture('Papers/Report.pdf').subarray(0, 5).toString('latin1')).toBe('%PDF-')
  })

  it('starts a ".png" file with the signature of a PNG', () => {
    expect(buildFixture('Pictures/Tree.png').subarray(0, 8).equals(PNG_SIGNATURE)).toBe(true)
  })

  it('gives bytes that are not text for any other extension', () => {
    const content = buildFixture('Videos/Trip.bin')
    expect(content.length).toBeGreaterThan(0)
    expect(content.includes(0)).toBe(true)
  })

  it('gives the same bytes for the same path and other bytes for another path', () => {
    expect(buildFixture('Papers/Report.pdf').equals(buildFixture('Papers/Report.pdf'))).toBe(true)
    expect(buildFixture('Papers/Report.pdf').equals(buildFixture('Papers/Plan.pdf'))).toBe(false)
  })

  it('writes exactly the number of megabytes requested', () => {
    expect(buildFixture('Videos/Trip.bin', 70).length).toBe(70 * 1024 * 1024)
  })

  it('keeps the first bytes of a bigger file the same as the smaller one of the same path', () => {
    const small = buildFixture('Videos/Trip.bin', 1)
    expect(buildFixture('Videos/Trip.bin', 3).subarray(0, small.length).equals(small)).toBe(true)
  })
})

describe('appendToFixture', () => {
  it('keeps the old bytes and adds the count of others', () => {
    const old = buildFixture('Papers/Report.pdf')

    const changed = appendToFixture(old, 100)

    expect(changed.length).toBe(old.length + 100)
    expect(changed.subarray(0, old.length).equals(old)).toBe(true)
  })

  it('appends the same bytes to the same file and other bytes to another', () => {
    const first = buildFixture('Papers/Report.pdf')
    expect(appendToFixture(first, 50).equals(appendToFixture(first, 50))).toBe(true)
    expect(appendToFixture(first, 50).equals(appendToFixture(buildFixture('Papers/Plan.pdf'), 50))).toBe(false)
  })
})

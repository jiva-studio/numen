import { describe, expect, it } from 'vitest'
import { formatBytes, formatBytesOf } from './bytes'

describe('formatBytes', () => {
  it('uses the largest unit that keeps the figure at one or more', () => {
    expect(formatBytes(0, 'en-US')).toBe('0 B')
    expect(formatBytes(1023, 'en-US')).toBe('1,023 B')
    expect(formatBytes(1024, 'en-US')).toBe('1.0 KB')
    expect(formatBytes(73400320, 'en-US')).toBe('70.0 MB')
    expect(formatBytes(5 * 1024 ** 3, 'en-US')).toBe('5.0 GB')
  })

  it('takes a negative size for zero', () => {
    expect(formatBytes(-5, 'en-US')).toBe('0 B')
  })
})

describe('formatBytesOf', () => {
  it('writes both figures in the unit of the whole', () => {
    expect(formatBytesOf(12 * 1024 ** 2, 41 * 1024 ** 2, 'en-US')).toBe('12 of 41 MB')
    expect(formatBytesOf(512 * 1024, 41 * 1024 ** 2, 'en-US')).toBe('0.5 of 41 MB')
    expect(formatBytesOf(10, 900, 'en-US')).toBe('10 of 900 B')
  })
})

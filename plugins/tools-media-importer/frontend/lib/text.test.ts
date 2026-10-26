import { describe, expect, it } from 'vitest'
import { formatClock, splitParagraphs } from './text'

describe('splitParagraphs', () => {
  it('keeps the lines that hold anything, trimmed', () => {
    expect(splitParagraphs('  First line.\n\n\n Second line. \n  \nThird.')).toEqual([
      'First line.',
      'Second line.',
      'Third.',
    ])
    expect(splitParagraphs('')).toEqual([])
  })
})

describe('formatClock', () => {
  it('writes minutes and seconds, and hours past an hour', () => {
    expect(formatClock(0)).toBe('0:00')
    expect(formatClock(1999)).toBe('0:01')
    expect(formatClock(67_000)).toBe('1:07')
    expect(formatClock(3_725_000)).toBe('1:02:05')
    expect(formatClock(-5)).toBe('0:00')
  })
})

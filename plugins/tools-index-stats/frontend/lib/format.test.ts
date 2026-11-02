import { describe, expect, it } from 'vitest'
import {
  formatBytes,
  formatCount,
  formatCounts,
  formatLabel,
  formatRate,
  formatTime,
} from './format'

describe('formatCount', () => {
  it('separates the thousands as the locale does', () => {
    expect(formatCount(12480, 'en-US')).toBe('12,480')
    expect(formatCount(12480, 'ru-RU')).toBe('12 480')
    expect(formatCount(12480, 'de-DE')).toBe('12.480')
  })

  it('writes small numbers and zero as they are', () => {
    expect(formatCount(0, 'en-US')).toBe('0')
    expect(formatCount(999, 'en-US')).toBe('999')
  })

  it('rounds a fraction away, since a count is whole', () => {
    expect(formatCount(1234.6, 'en-US')).toBe('1,235')
  })
})

describe('formatLabel', () => {
  it('shows the vectors alone once nothing waits', () => {
    expect(formatLabel({ vectors: 12480, chunks: 12480, waiting: 0 }, 'ru-RU')).toBe('⬡ 12 480')
  })

  it('shows the vectors over the chunks that should carry one while embedding runs', () => {
    expect(formatLabel({ vectors: 12480, chunks: 13100, waiting: 620 }, 'ru-RU')).toBe(
      '⬡ 12 480 / 13 100',
    )
  })

  it('shows zero for an empty index', () => {
    expect(formatLabel({ vectors: 0, chunks: 0, waiting: 0 }, 'en-US')).toBe('⬡ 0')
  })
})

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

describe('formatRate', () => {
  it('keeps one decimal', () => {
    expect(formatRate(18.46, 'en-US')).toBe('18.5')
    expect(formatRate(0, 'en-US')).toBe('0')
  })
})

describe('formatTime', () => {
  it('writes the time of day of an instant', () => {
    expect(formatTime('2026-11-02T12:00:07Z', 'en-US')).toMatch(/^\d{1,2}:\d{2}:\d{2}/)
  })

  it('gives nothing for a text that is no instant', () => {
    expect(formatTime('later', 'en-US')).toBe('')
  })
})

describe('formatCounts', () => {
  it('writes the vectors alone, or over the chunks while some wait', () => {
    expect(formatCounts({ vectors: 12480, chunks: 12480, waiting: 0 }, 'en-US')).toBe('12,480')
    expect(formatCounts({ vectors: 12480, chunks: 13100, waiting: 620 }, 'en-US')).toBe(
      '12,480 / 13,100',
    )
  })
})

import { describe, expect, it } from 'vitest'
import {
  formatBytes,
  formatCount,
  formatSummary,
  formatVectors,
  isPending,
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

describe('formatVectors', () => {
  it('writes the vectors alone, whether or not chunks wait', () => {
    expect(formatVectors({ vectors: 12480 }, 'ru-RU')).toBe(formatCount(12480, 'ru-RU'))
    expect(formatVectors({ vectors: 124507 }, 'en-US')).toBe('124,507')
    expect(formatVectors({ vectors: 0 }, 'en-US')).toBe('0')
  })

  it('never writes a chunk count', () => {
    for (const waiting of [0, 1, 620]) {
      const stats = { vectors: 124507, chunks: 124508, waiting }
      expect(formatVectors(stats, 'en-US')).not.toContain('/')
    }
  })
})

describe('isPending', () => {
  it('is true while chunks wait for a vector', () => {
    expect(isPending({ waiting: 1 })).toBe(true)
    expect(isPending({ waiting: 620 })).toBe(true)
  })

  it('is false when the queue is empty', () => {
    expect(isPending({ waiting: 0 })).toBe(false)
  })
})

describe('formatSummary', () => {
  it('counts the vectors, and says that indexing runs while chunks wait', () => {
    expect(formatSummary({ vectors: 124507, waiting: 1 }, 'en-US')).toBe(
      '124,507 vectors, indexing',
    )
    expect(formatSummary({ vectors: 124508, waiting: 0 }, 'en-US')).toBe('124,508 vectors')
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

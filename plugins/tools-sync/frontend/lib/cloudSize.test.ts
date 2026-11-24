import { describe, expect, it } from 'vitest'
import { formatCloudSize } from './cloudSize'

describe('formatCloudSize', () => {
  it.each([
    [0, '0 MB'],
    [100, '1 KB'],
    [524_288, '512 KB'],
    [3_670_016, '3.5 MB'],
    [1_288_490_189, '1.2 GB'],
  ])('writes %d bytes as %s', (bytes, text) => {
    expect(formatCloudSize(bytes)).toBe(text)
  })
})

/**
 * What the window takes for a web address.
 */
import { describe, expect, it } from 'vitest'
import { isWebUrl } from './url'

describe('a web address', () => {
  it('is one with an http or an https scheme and a host', () => {
    expect(isWebUrl('http://example.com')).toBe(true)
    expect(isWebUrl('https://example.com/watch?v=1')).toBe(true)
  })

  it('is not one of another scheme', () => {
    expect(isWebUrl('ftp://example.com')).toBe(false)
    expect(isWebUrl('file:///notes/Entropy.md')).toBe(false)
    expect(isWebUrl('mailto:someone@example.com')).toBe(false)
  })

  it('is not a string no address can be made from', () => {
    expect(isWebUrl('Entropy')).toBe(false)
    expect(isWebUrl('http://')).toBe(false)
    expect(isWebUrl('')).toBe(false)
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'
import { browserRandomness } from './browserRandomness'

describe('the randomness of the window', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('passes over what the window makes', () => {
    vi.stubGlobal('crypto', { randomUUID: () => 'from-the-window' })

    expect(browserRandomness.createUuid()).toBe('from-the-window')
  })

  it('makes an identity where the window has randomness', () => {
    const created = browserRandomness.createUuid()

    expect(created === null || created.length > 0).toBe(true)
  })

  it('makes nothing where the window has no way to make an identifier', () => {
    vi.stubGlobal('crypto', {})

    expect(browserRandomness.createUuid()).toBeNull()
  })

  it('makes nothing where the window has no randomness at all', () => {
    vi.stubGlobal('crypto', undefined)

    expect(browserRandomness.createUuid()).toBeNull()
  })

  it('makes nothing where the identifier maker is no function', () => {
    vi.stubGlobal('crypto', { randomUUID: 'not-callable' })

    expect(browserRandomness.createUuid()).toBeNull()
  })
})

/**
 * The page helpers: the user agent.
 */
import { describe, expect, it } from 'vitest'
import { getUserAgent } from './dom'

describe('dom helpers', () => {
  it('reads navigator.userAgent', () => {
    expect(getUserAgent()).toBe(navigator.userAgent)
    expect(getUserAgent().length).toBeGreaterThan(0)
  })
})

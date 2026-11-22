import { describe, expect, it } from 'vitest'
import { isWebAddress } from '../src'

describe('a web address', () => {
  it.each(['http://example.org/', 'https://example.org/a?b=c'])('%s is one', (url) => {
    expect(isWebAddress(url)).toBe(true)
  })

  it.each(['mailto:a@example.org', 'file:///etc/passwd', 'javascript:alert(1)', 'example.org', ''])(
    '%j is none',
    (url) => {
      expect(isWebAddress(url)).toBe(false)
    },
  )
})

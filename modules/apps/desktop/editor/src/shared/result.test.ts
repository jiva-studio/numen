import { describe, expect, it } from 'vitest'
import { err, ok, type Result } from './result'

describe('Result pattern', () => {
  it('creates a successful result carrying a value', () => {
    const res: Result<string> = ok('success')

    expect(res.ok).toBe(true)
    if (res.ok) {
      expect(res.value).toBe('success')
    }
  })

  it('creates a failure result carrying an error discriminant', () => {
    const res: Result<string> = err('missing')

    expect(res.ok).toBe(false)
    if (!res.ok) {
      expect(res.error).toBe('missing')
    }
  })
})

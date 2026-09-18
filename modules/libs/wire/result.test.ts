/**
 * The two cases a call can answer with, and nothing between them.
 *
 * A caller reads `ok` and the other field is there only in that case, which is
 * what the compiler is asked to hold.
 */
import { describe, expect, it } from 'vitest'
import { asFailure, asValue, type Result } from './result'

describe('what a call that can fail answers with', () => {
  it('carries the value where it came back', () => {
    const answer: Result<number> = asValue(7)

    expect(answer.ok).toBe(true)
    expect(answer.ok ? answer.value : null).toBe(7)
  })

  it('carries why, where nothing came back', () => {
    const answer: Result<number> = asFailure('the folder has gone')

    expect(answer.ok).toBe(false)
    expect(answer.ok ? null : answer.error).toBe('the folder has gone')
  })

  it('carries a value of nothing without reading as a failure', () => {
    const answer: Result<string> = asValue('')

    expect(answer.ok).toBe(true)
  })
})

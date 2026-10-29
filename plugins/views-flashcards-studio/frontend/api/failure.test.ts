import { describe, expect, it } from 'vitest'
import { ApiFailure, CANCELED, formatFailureMessage } from './failure'

const fail = (code: string, message: string) => new ApiFailure({ code, message })

describe('the words of a failure', () => {
  it('say the message of a code written for the person', () => {
    expect(formatFailureMessage(fail('failed_precondition', ' the vault is closing '))).toBe(
      'the vault is closing',
    )
  })

  it('say the words of a code whose message is not written for the person', () => {
    expect(formatFailureMessage(fail('not_found', 'open /x: no such file'))).toBe('not found')
    expect(formatFailureMessage(fail('internal', 'EIO'))).toBe('internal error')
  })

  it('say nothing for a call the window stopped', () => {
    expect(formatFailureMessage(new ApiFailure(CANCELED))).toBe('')
  })

  it('leave any other fault to the SDK', () => {
    expect(formatFailureMessage(new TypeError('x is not a function'))).toMatch(/\S/)
  })
})

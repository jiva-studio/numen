import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { ErrorCode as ProtoErrorCode, ErrorCodeSchema } from '@numen/protocol'

import { formatErrorCodeMessage, formatErrorMessage } from './error'

/** Past every error code the schema names today. */
const UNNAMED_CODE = 999 as ProtoErrorCode

/** Past every code connect's own enum names today. */
const UNNAMED_CONNECT_CODE = 999 as Code

describe('an error code', () => {
  it.each(ErrorCodeSchema.values.map((value) => [value.name, value.number] as const))(
    'has words of its own for %s',
    (name, number) => {
      expect(
        formatErrorCodeMessage(number as ProtoErrorCode),
        `${name} is drawn as nothing`,
      ).toMatch(/\S/)
    },
  )

  it('says none of them the same way as another', () => {
    const formatted = ErrorCodeSchema.values.map((value) =>
      formatErrorCodeMessage(value.number as ProtoErrorCode),
    )
    expect(new Set(formatted).size).toBe(formatted.length)
  })

  it('says nothing where there was no error', () => {
    expect(formatErrorCodeMessage(undefined)).toBe('')
  })

  it('says something plain where the code is empty here yet', () => {
    expect(formatErrorCodeMessage(UNNAMED_CODE)).toMatch(/\S/)
  })
})

describe('a fault', () => {
  it('says what the person can do about a connection that has gone', () => {
    expect(formatErrorMessage(new ConnectError('read /notes: EIO', Code.Unknown))).toBe(
      'numen did not answer, so nothing was done — it may have stopped, and the window keeps trying',
    )
  })

  it('carries no word of what was thrown, where the code wraps a Go error', () => {
    const message = formatErrorMessage(
      new ConnectError('sql: no rows in result set', Code.Internal),
    )
    expect(message).not.toContain('sql')
  })

  it('says the sentence the application wrote, where it wrote one', () => {
    const message = formatErrorMessage(new ConnectError('request error', Code.FailedPrecondition))
    expect(message).toBe('request error')
  })

  it('falls back on the code where the application sent no sentence', () => {
    expect(formatErrorMessage(new ConnectError('', Code.Unavailable))).toBe(
      'numen is not answering just now — try again in a moment',
    )
  })

  it('says nothing the file it could not find is called', () => {
    const message = formatErrorMessage(
      new ConnectError('stat notes/Leaf mould.md: no such file', Code.NotFound),
    )
    expect(message).toBe('what was asked for is not there')
  })

  it('says nothing at all about a call the window itself stopped', () => {
    expect(formatErrorMessage(new ConnectError('stopped', Code.Canceled))).toBe('')
  })

  it('takes what was never a connect error, and what was never an error', () => {
    expect(formatErrorMessage(new TypeError('undefined is not a function'))).toMatch(/\S/)
    expect(formatErrorMessage('nothing in particular')).toMatch(/\S/)
    expect(formatErrorMessage(undefined)).toMatch(/\S/)
  })

  it('says something plain where the code is empty here yet', () => {
    const message = formatErrorMessage(new ConnectError('', UNNAMED_CONNECT_CODE))
    expect(message).toMatch(/\S/)
  })
})

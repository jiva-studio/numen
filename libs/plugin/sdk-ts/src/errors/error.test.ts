import { runInNewContext } from 'node:vm'
import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import type { ErrorCode as ProtoErrorCode } from '@numen/protocol'
import { ErrorCodeSchema } from '@numen/protocol'

import { ReportedError, formatErrorCodeMessage, formatErrorMessage } from './error'

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
      'Numen is not responding; retrying',
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
      'Numen is busy; try again',
    )
  })

  it('says nothing the file it could not find is called', () => {
    const message = formatErrorMessage(
      new ConnectError('stat notes/Leaf mould.md: no such file', Code.NotFound),
    )
    expect(message).toBe('not found')
  })

  it('says nothing at all about a call the window itself stopped', () => {
    expect(formatErrorMessage(new ConnectError('stopped', Code.Canceled))).toBe('')
  })

  it('takes what was never a connect error, and what was never an error', () => {
    expect(formatErrorMessage(new TypeError('undefined is not a function'))).toMatch(/\S/)
    expect(formatErrorMessage('nothing in particular')).toMatch(/\S/)
    expect(formatErrorMessage(undefined)).toMatch(/\S/)
  })

  it('shows the text of a ReportedError', () => {
    expect(
      formatErrorMessage(new ReportedError('The deck sample.deck could not be read: no cards')),
    ).toBe('The deck sample.deck could not be read: no cards')
  })

  it('shows the text of an error named ReportedError from the core-client copy', () => {
    const reported = Object.assign(new Error('The deck sample.deck could not be read: no cards'), {
      name: 'ReportedError',
    })
    expect(formatErrorMessage(reported)).toBe('The deck sample.deck could not be read: no cards')
  })

  it('shows the text of a ReportedError built in another realm', () => {
    const reported: unknown = runInNewContext(
      'const error = new Error(text); error.name = "ReportedError"; error',
      { text: 'The deck sample.deck could not be read: no cards' },
    )
    expect(reported).not.toBeInstanceOf(Error)
    expect(formatErrorMessage(reported)).toBe('The deck sample.deck could not be read: no cards')
  })

  it('shows the generic text for a ReportedError with blank text', () => {
    expect(formatErrorMessage(new ReportedError('  '))).toBe('Numen is not responding; retrying')
  })

  it('shows the generic text for null', () => {
    expect(formatErrorMessage(null)).toBe('Numen is not responding; retrying')
  })

  it('shows the generic text for an object named ReportedError whose message is not text', () => {
    expect(formatErrorMessage({ name: 'ReportedError', message: 42 })).toBe(
      'Numen is not responding; retrying',
    )
  })

  it('shows the generic text for an Error that is not a ReportedError', () => {
    const failure = new Error('open /Users/someone/Vault/library/Broken.epub: permission denied')
    expect(formatErrorMessage(failure)).toBe('Numen is not responding; retrying')
  })

  it('shows the generic text for a TypeError thrown by fetch outside connect', () => {
    expect(formatErrorMessage(new TypeError('Load failed'))).toBe(
      'Numen is not responding; retrying',
    )
  })

  it('shows the generic text for a native error with its own code field', () => {
    const failure = Object.assign(
      new Error('mkdir /data/user/0/md.numen.sample/files/vault: permission denied'),
      { code: 'UNAVAILABLE' },
    )
    expect(formatErrorMessage(failure)).toBe('Numen is not responding; retrying')
  })

  it('shows the generic text for a ConnectError from a failed fetch', () => {
    const dropped = ConnectError.from(new TypeError('Failed to fetch'))
    expect(formatErrorMessage(dropped)).toBe('Numen is not responding; retrying')
  })

  it('shows the busy text for Unavailable with an empty message', () => {
    expect(formatErrorMessage(new ConnectError('', Code.Unavailable))).toBe(
      'Numen is busy; try again',
    )
  })

  it('returns empty text for an AbortError', () => {
    expect(formatErrorMessage(new DOMException('aborted', 'AbortError'))).toBe('')
  })

  it('shows the generic text for an Error with blank text', () => {
    expect(formatErrorMessage(new Error('  '))).toBe('Numen is not responding; retrying')
  })

  it('says something plain where the code is empty here yet', () => {
    const message = formatErrorMessage(new ConnectError('', UNNAMED_CONNECT_CODE))
    expect(message).toMatch(/\S/)
  })
})

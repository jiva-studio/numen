import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { Refusal, RefusalSchema } from '@numen/protocol'

import { refusalWords, troubleWords } from './trouble'

describe('a refusal', () => {
  // The list walked here is the descriptor the generated code carries, so the
  // schema is the only list of refusals there is: one added to it arrives
  // wordless and fails, rather than being drawn as nothing.
  it.each(RefusalSchema.values.map((value) => [value.name, value.number] as const))(
    'has words of its own for %s',
    (name, number) => {
      expect(refusalWords(number as Refusal), `${name} is drawn as nothing`).toMatch(/\S/)
    },
  )

  it('says none of them the same way as another', () => {
    const said = RefusalSchema.values.map((value) => refusalWords(value.number as Refusal))
    expect(new Set(said).size).toBe(said.length)
  })

  it('says nothing where the answer was not refused', () => {
    expect(refusalWords(undefined)).toBe('')
  })
})

describe('a fault', () => {
  it('says what the person can do about a connection that has gone', () => {
    expect(troubleWords(new ConnectError('read /notes: EIO', Code.Unknown))).toBe(
      'numen did not answer, so nothing was done — it may have stopped, and the window keeps trying',
    )
  })

  it('carries no word of what was thrown, where the code wraps a Go error', () => {
    const said = troubleWords(new ConnectError('sql: no rows in result set', Code.Internal))
    expect(said).not.toContain('sql')
  })

  it('says the sentence the application wrote, where it wrote one', () => {
    const said = troubleWords(
      new ConnectError('this recording is being listened to', Code.FailedPrecondition),
    )
    expect(said).toBe('this recording is being listened to')
  })

  it('falls back on the code where the application sent no sentence', () => {
    expect(troubleWords(new ConnectError('', Code.Unavailable))).toBe(
      'numen is not answering just now — try again in a moment',
    )
  })

  it('says nothing the file it could not find is called', () => {
    const said = troubleWords(
      new ConnectError('stat notes/Leaf mould.md: no such file', Code.NotFound),
    )
    expect(said).toBe('what was asked for is not there')
  })

  it('says nothing at all about a call the window itself stopped', () => {
    expect(troubleWords(new ConnectError('stopped', Code.Canceled))).toBe('')
  })

  it('takes what was never a connect error, and what was never an error', () => {
    expect(troubleWords(new TypeError('undefined is not a function'))).toMatch(/\S/)
    expect(troubleWords('nothing in particular')).toMatch(/\S/)
    expect(troubleWords(undefined)).toMatch(/\S/)
  })
})

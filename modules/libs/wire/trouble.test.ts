import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { Refusal, RefusalSchema } from '@numen/protocol'

import { refused, sentence } from './trouble'

describe('a refusal', () => {
  // The list walked here is the descriptor the generated code carries, so the
  // schema is the only list of refusals there is: one added to it arrives
  // wordless and fails, rather than being drawn as nothing.
  it.each(RefusalSchema.values.map((value) => [value.name, value.number] as const))(
    'has words of its own for %s',
    (name, number) => {
      const said = refused(number as Refusal)
      expect(said, `${name} is drawn as nothing`).toMatch(/\S/)
    },
  )

  it('says none of them the same way as another', () => {
    const said = RefusalSchema.values.map((value) => refused(value.number as Refusal))
    expect(new Set(said).size).toBe(said.length)
  })

  it('says nothing where the answer was not refused', () => {
    expect(refused(undefined)).toBe('')
  })
})

describe('a fault', () => {
  it('says what the person can do about a connection that has gone', () => {
    expect(sentence(new ConnectError('read /notes: EIO', Code.Unknown))).toBe(
      'Nothing was done: numen did not answer. It may have stopped, and the window keeps trying.',
    )
  })

  it('carries no word of what was thrown', () => {
    const said = sentence(new ConnectError('sql: no rows in result set', Code.Internal))
    expect(said).not.toContain('sql')
  })

  it('says nothing at all about a call the window itself stopped', () => {
    expect(sentence(new ConnectError('stopped', Code.Canceled))).toBe('')
  })

  it('takes what was never a connect error, and what was never an error', () => {
    expect(sentence(new TypeError('undefined is not a function'))).toMatch(/\S/)
    expect(sentence('nothing in particular')).toMatch(/\S/)
    expect(sentence(undefined)).toMatch(/\S/)
  })
})

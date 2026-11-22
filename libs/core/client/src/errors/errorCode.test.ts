import { describe, expect, it } from 'vitest'
import { ErrorCodeSchema } from '@numen/protocol'
import { formatErrorCodeMessage } from '@numen/result'

describe('an error code of the schema', () => {
  it.each(ErrorCodeSchema.values.map((value) => [value.name, value.number] as const))(
    'has words of its own for %s',
    (name, number) => {
      expect(formatErrorCodeMessage(number), `${name} is drawn as nothing`).not.toBe('internal error')
    },
  )

  it('is worded differently from every other', () => {
    const formatted = ErrorCodeSchema.values.map((value) => formatErrorCodeMessage(value.number))
    expect(new Set(formatted).size).toBe(formatted.length)
  })
})

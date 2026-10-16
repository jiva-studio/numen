import { describe, expect, it } from 'vitest'
import { createMediaKinds } from './media'

describe('createMediaKinds', () => {
  it('returns empty object when media kinds are provided via plugins', () => {
    const kinds = createMediaKinds({} as any)
    expect(kinds).toStrictEqual({})
  })
})

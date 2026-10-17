/** What the reader kinds are handed. */
import { describe, expect, it } from 'vitest'
import { createReaderKinds, type ReaderKindsDeps } from './readers'

describe('the reader kinds', () => {
  it('returns empty kinds since readers are provided by plugins', () => {
    const kinds = createReaderKinds({
      tabOpeners: {},
      held: { handle: {} },
    } as unknown as ReaderKindsDeps)

    expect(kinds).toEqual({})
  })
})

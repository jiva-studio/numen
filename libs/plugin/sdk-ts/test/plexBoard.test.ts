import { describe, expectTypeOf, it } from 'vitest'
import type { PlexBoardViewProps } from '../src'

describe('the props of a view of the board of the plex graph', () => {
  it('are the board key, the path in front and go', () => {
    expectTypeOf<PlexBoardViewProps>().toEqualTypeOf<{
      readonly board: string
      readonly path: string
      readonly go: (path: string) => void
    }>()
  })
})

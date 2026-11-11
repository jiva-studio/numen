/** What is read out where a group has nothing to show. */
import { describe, expect, it } from 'vitest'
import { usePalettePlaces } from './places'
import type { PaletteAction, PaletteGroup, PaletteItem } from '../lib/item'

const OPEN: PaletteAction = { id: 'open', text: 'Open note' }

const item = (id: string, isDisabled = false): PaletteItem => ({
  id,
  title: id,
  actions: [OPEN],
  isDisabled,
})

const group = (
  id: string,
  items: PaletteItem[],
  rest: Partial<PaletteGroup> = {},
): PaletteGroup => ({
  id,
  title: id,
  items,
  ...rest,
})

const setUp = (groups: PaletteGroup[]) => usePalettePlaces({ groups: () => groups })

describe('usePalettePlaces', () => {
  describe('what is read out', () => {
    it('is what the groups with nothing say, in the order they stand', () => {
      const state = setUp([
        group('seen', [item('a')], { silence: 'unheard' }),
        group('first', [], { silence: 'No names' }),
        group('loading', [], { isWorking: true, silence: 'Still looking' }),
        group('second', [], { silence: 'No notes' }),
        group('quiet', [item('b')]),
      ])
      expect(state.announcement.value).toBe('No names. No notes')
    })

    it('is one group alone as it says it', () => {
      const state = setUp([group('first', [], { silence: 'No names' })])
      expect(state.announcement.value).toBe('No names')
    })

    it('is empty where no group is silent', () => {
      const state = setUp([group('seen', [item('a')]), group('loading', [], { isWorking: true })])
      expect(state.announcement.value).toBe('')
    })
  })
})

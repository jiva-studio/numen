/**
 * What the keyboard relies on as answers arrive, as a pointer crosses the list
 * and as a question is typed, and what is read out where a group has nothing.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { usePalettePlaces } from './places'
import type { PaletteAction, PaletteGroup, PaletteItem } from '../lib/item'

const OPEN: PaletteAction = { id: 'open', text: 'Open the note' }

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

const setUp = (initial: PaletteGroup[]) => {
  const groups = ref<PaletteGroup[]>(initial)
  const searchQuery = ref('')
  const onItemActive = vi.fn<(item: string) => void>()
  const tellChoice = vi.fn<(item: string, action: string) => void>()
  const state = usePalettePlaces({ groups: () => groups.value, typed: searchQuery, onItemActive, tellChoice })
  return { groups, typed: searchQuery, onItemActive, tellChoice, state }
}

const move = (x: number, y: number): PointerEvent => ({ clientX: x, clientY: y }) as PointerEvent

describe('usePalettePlaces', () => {
  describe('what is read out', () => {
    it('is what the groups with nothing say, in the order they stand', () => {
      const { state } = setUp([
        group('seen', [item('a')], { silence: 'unheard' }),
        group('first', [], { silence: 'No names' }),
        group('loading', [], { isWorking: true, silence: 'Still looking' }),
        group('second', [], { silence: 'No notes' }),
        group('quiet', [item('b')]),
      ])
      expect(state.announcement.value).toBe('No names. No notes')
    })

    it('is one group alone as it says it', () => {
      const { state } = setUp([group('first', [], { silence: 'No names' })])
      expect(state.announcement.value).toBe('No names')
    })

    it('is empty where no group is silent', () => {
      const { state } = setUp([
        group('seen', [item('a')]),
        group('loading', [], { isWorking: true }),
      ])
      expect(state.announcement.value).toBe('')
    })
  })

  describe('where the keyboard lands as answers arrive', () => {
    it('lands on the first item that can be chosen', async () => {
      const { groups, state } = setUp([])
      groups.value = [group('g', [item('a'), item('b'), item('c')])]
      await nextTick()
      expect(state.here.value).toBe(0)
      expect(state.activeItem.value?.id).toBe('a')
    })

    it('stays on the item it is on while the list changes', async () => {
      const { groups, state } = setUp([group('g', [item('a'), item('b')])])
      await nextTick()
      state.goTo(1)
      groups.value = [group('g', [item('a'), item('b'), item('c')])]
      await nextTick()
      expect(state.activeItem.value?.id).toBe('b')
    })

    it('stays on the first row when that row is the one it holds', async () => {
      const { groups, state } = setUp([group('g', [item('a', true), item('b')])])
      state.selectedId.value = 'a'
      groups.value = [group('g', [item('a', true), item('b'), item('c')])]
      await nextTick()
      expect(state.here.value).toBe(0)
    })

    it('says at its current position whenever it moves', async () => {
      const { onItemActive, state } = setUp([group('g', [item('a'), item('b')])])
      state.goTo(1)
      await nextTick()
      expect(onItemActive).toHaveBeenLastCalledWith('b')
    })

    it('is put on nothing by a fresh question', async () => {
      const { typed, onItemActive, state } = setUp([group('g', [item('a'), item('b')])])
      state.goTo(1)
      await nextTick()
      typed.value = 'n'
      await nextTick()
      expect(state.selectedId.value).toBe('')
      expect(state.activeItem.value).toBeUndefined()
      expect(onItemActive).toHaveBeenLastCalledWith('')
    })
  })

  describe('a pointer crossing the list', () => {
    const list = () => setUp([group('g', [item('a'), item('b'), item('c', true)])])

    it('lights the row it first moves over', () => {
      const { state } = list()
      state.lightAt(1, move(5, 5))
      expect(state.activeItem.value?.id).toBe('b')
    })

    it('does not light a row for a pointer that has never been seen to move', () => {
      const { state } = list()
      state.lightAt(1, move(-1, -1))
      expect(state.here.value).toBe(-1)
    })

    it('lights nothing for a pointer standing still', () => {
      const { state } = list()
      state.lightAt(1, move(5, 5))
      state.goTo(0)
      state.lightAt(1, move(5, 5))
      expect(state.activeItem.value?.id).toBe('a')
    })

    it('lights a row when only one coordinate has moved', () => {
      const { state } = list()
      state.lightAt(0, move(5, 5))
      state.lightAt(1, move(5, 6))
      expect(state.activeItem.value?.id).toBe('b')
      state.lightAt(0, move(6, 6))
      expect(state.activeItem.value?.id).toBe('a')
    })

    it('passes over a row that cannot be chosen', () => {
      const { state } = list()
      state.goTo(0)
      state.lightAt(2, move(5, 5))
      expect(state.activeItem.value?.id).toBe('a')
    })

    it('ignores a row that is not in the list', () => {
      const { state } = list()
      state.goTo(0)
      expect(() => state.lightAt(9, move(5, 5))).not.toThrow()
      expect(state.activeItem.value?.id).toBe('a')
    })
  })

  describe('choosing', () => {
    it('returns the item and the action Enter or Shift and Enter reach', () => {
      const { tellChoice, state } = setUp([
        group('g', [{ ...item('a'), actions: [OPEN, { id: 'travel', text: 'Show' }] }]),
      ])
      state.chooseAt(0, false)
      state.chooseAt(0, true)
      expect(tellChoice.mock.calls).toEqual([
        ['a', 'open'],
        ['a', 'travel'],
      ])
    })

    it('chooses nothing at a row that is not there', () => {
      const { tellChoice, state } = setUp([group('g', [item('a')])])
      state.chooseAt(4, false)
      expect(tellChoice).not.toHaveBeenCalled()
    })
  })
})

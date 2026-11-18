/** The order of the rows is frozen for one opening of the palette: the next opening computes it afresh. */
import { describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import type { PaletteMultiProps } from '../lib/multi'
import { useMultiPick } from './multiPick'

type Item = PaletteMultiProps['items'][number]

const TREES: readonly Item[] = [
  { id: 'birch', title: 'Birch', state: 'unset' },
  { id: 'cedar', title: 'Cedar', state: 'mixed' },
  { id: 'alder', title: 'Alder', state: 'unset' },
]

const makeMulti = (items: readonly Item[]): PaletteMultiProps => ({
  title: 'Trees',
  placeholder: 'Tree',
  items,
  recent: [],
  getCreateTitle: (text) => `Create ${text}`,
})

describe('useMultiPick reopening', () => {
  it('orders the rows afresh when the palette is closed and opened again', () => {
    const props = reactive<{ groups: never[]; isOpen: boolean; multi: PaletteMultiProps }>({
      groups: [],
      isOpen: true,
      multi: makeMulti(TREES),
    })
    const pick = useMultiPick(props, ref(''), vi.fn() as never)
    const getTitles = () => (pick.groups.value[0]?.items ?? []).map((one) => one.title)
    expect(getTitles()).toStrictEqual(['Cedar', 'Alder', 'Birch'])

    props.multi = makeMulti(
      TREES.map((one) => (one.id === 'birch' ? { ...one, state: 'set' } : one)),
    )
    expect(getTitles()).toStrictEqual(['Cedar', 'Alder', 'Birch'])

    props.isOpen = false
    props.isOpen = true
    expect(getTitles()).toStrictEqual(['Birch', 'Cedar', 'Alder'])
  })
})

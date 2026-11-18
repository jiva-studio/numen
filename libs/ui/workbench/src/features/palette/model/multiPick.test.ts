/**
 * The rows of the multiple step keep the order they had when the step opened. A toggle changes the check
 * of a row and nothing else, so the row a person pressed stays where the pointer is.
 */
import { describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import type { PaletteMultiProps } from '../lib/multi'
import { useMultiPick } from './multiPick'

type State = 'set' | 'mixed' | 'unset'
type Item = PaletteMultiProps['items'][number]

const TREES: readonly Item[] = [
  { id: 'birch', title: 'Birch', state: 'unset' },
  { id: 'willow', title: 'Willow', state: 'unset' },
  { id: 'cedar', title: 'Cedar', state: 'mixed' },
  { id: 'alder', title: 'Alder', state: 'unset' },
  { id: 'maple', title: 'Maple', state: 'unset' },
  { id: 'fir', title: 'Fir', state: 'set' },
]

const makeMulti = (
  items: readonly Item[] = TREES,
  recent: readonly string[] = ['fir', 'willow', 'maple'],
): PaletteMultiProps => ({
  title: 'Trees',
  placeholder: 'Tree',
  items,
  recent,
  getCreateTitle: (text) => `Create ${text}`,
})

const withStates = (states: Record<string, State>, items = TREES): readonly Item[] =>
  items.map((one) => ({ ...one, state: states[one.id] ?? one.state }))

const setUp = (multi: PaletteMultiProps = makeMulti()) => {
  const props = reactive<{ groups: never[]; multi: PaletteMultiProps }>({ groups: [], multi })
  const typed = ref('')
  const emit = vi.fn()
  const pick = useMultiPick(props, typed, emit as never)
  const getTitles = () => (pick.groups.value[0]?.items ?? []).map((one) => one.title)
  return { props, typed, emit, pick, getTitles }
}

const OPENED = ['Cedar', 'Fir', 'Willow', 'Maple', 'Alder', 'Birch']

describe('useMultiPick order', () => {
  it('opens with the set and mixed rows first, the recent ones next and the rest by title', () => {
    const { getTitles } = setUp()
    expect(getTitles()).toStrictEqual(OPENED)
  })

  it('keeps the order when a row that was not set is set', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti(withStates({ maple: 'set' }))
    expect(getTitles()).toStrictEqual(OPENED)
  })

  it('keeps the order when a row that was set is unset', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti(withStates({ fir: 'unset', cedar: 'unset' }))
    expect(getTitles()).toStrictEqual(OPENED)
  })

  it('keeps the order through a toggle and the toggle back', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti(withStates({ alder: 'set' }))
    expect(getTitles()).toStrictEqual(OPENED)
    props.multi = makeMulti(withStates({ alder: 'unset' }))
    expect(getTitles()).toStrictEqual(OPENED)
  })

  it('shows the new state on the row that was toggled', () => {
    const { props, pick, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti(withStates({ alder: 'set' }))
    const alder = pick.groups.value[0]?.items.find((one) => one.id === 'alder')
    expect(alder?.state).toBe('set')
  })

  it('keeps the order when the caller sends the same items in another order', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti([...TREES].reverse())
    expect(getTitles()).toStrictEqual(OPENED)
  })

  it('keeps the order when the recent ids change', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti(TREES, ['maple', 'alder'])
    expect(getTitles()).toStrictEqual(OPENED)
  })

  it('narrows the rows to the typed words and keeps their relative order', () => {
    const { typed, getTitles } = setUp()
    getTitles()
    typed.value = 'a'
    expect(getTitles().filter((title) => !title.startsWith('Create'))).toStrictEqual([
      'Cedar',
      'Maple',
      'Alder',
    ])
  })

  it('keeps the order of the narrowed rows after a toggle among them', () => {
    const { props, typed, getTitles } = setUp()
    getTitles()
    typed.value = 'a'
    props.multi = makeMulti(withStates({ alder: 'set', cedar: 'unset' }))
    expect(getTitles().filter((title) => !title.startsWith('Create'))).toStrictEqual([
      'Cedar',
      'Maple',
      'Alder',
    ])
  })

  it('brings back every row in the opening order when the typed text is cleared', () => {
    const { props, typed, getTitles } = setUp()
    getTitles()
    typed.value = 'a'
    props.multi = makeMulti(withStates({ alder: 'set' }))
    getTitles()
    typed.value = ''
    expect(getTitles()).toStrictEqual(OPENED)
  })
})

describe('useMultiPick items the caller adds', () => {
  const ACACIA: Item = { id: 'acacia', title: 'Acacia', state: 'unset' }

  it('puts a new item at the end', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti([ACACIA, ...TREES])
    expect(getTitles()).toStrictEqual([...OPENED, 'Acacia'])
  })

  it('puts new items at the end in the order the caller gives them', () => {
    const { props, getTitles } = setUp()
    getTitles()
    const aspen: Item = { id: 'aspen', title: 'Aspen', state: 'unset' }
    props.multi = makeMulti([...TREES, ACACIA, aspen])
    expect(getTitles()).toStrictEqual([...OPENED, 'Acacia', 'Aspen'])
  })

  it('keeps a new item at the end when it is set, and after the next value of items', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti([...TREES, ACACIA])
    getTitles()
    props.multi = makeMulti([...TREES, { ...ACACIA, state: 'set' }])
    expect(getTitles()).toStrictEqual([...OPENED, 'Acacia'])
  })

  it('restores the place of an item that is removed and added again', () => {
    const { props, getTitles } = setUp()
    getTitles()
    props.multi = makeMulti(TREES.filter((one) => one.id !== 'maple'))
    expect(getTitles()).toStrictEqual(['Cedar', 'Fir', 'Willow', 'Alder', 'Birch'])
    props.multi = makeMulti(TREES)
    expect(getTitles()).toStrictEqual(OPENED)
  })
})

describe('useMultiPick row that creates an item', () => {
  const HARBOUR: Item = { id: 'harbour', title: 'Harbour', state: 'set' }

  it('is ahead of the rows found while the typed text names no item', () => {
    const { typed, getTitles } = setUp()
    typed.value = 'a'
    expect(getTitles()[0]).toBe('Create a')
  })

  it('leaves the new item first after the caller adds it, and the typed text is cleared', () => {
    const { props, typed, pick, getTitles } = setUp()
    typed.value = 'Harbour'
    pick.tellChoice('\u0000create', 'create')
    typed.value = ''
    props.multi = makeMulti([...TREES, HARBOUR])
    expect(getTitles()).toStrictEqual(['Harbour', ...OPENED])
  })

  it('keeps the created item first through a toggle of it and of another row', () => {
    const { props, typed, pick, getTitles } = setUp()
    typed.value = 'Harbour'
    pick.tellChoice('\u0000create', 'create')
    typed.value = ''
    props.multi = makeMulti([...TREES, HARBOUR])
    getTitles()
    props.multi = makeMulti(
      withStates({ maple: 'set' }, [...TREES, { ...HARBOUR, state: 'unset' }]),
    )
    expect(getTitles()).toStrictEqual(['Harbour', ...OPENED])
  })

  it('puts an item that the person did not create at the end even when one was created', () => {
    const { props, typed, pick, getTitles } = setUp()
    getTitles()
    typed.value = 'Harbour'
    pick.tellChoice('\u0000create', 'create')
    typed.value = ''
    props.multi = makeMulti([...TREES, HARBOUR, { id: 'acacia', title: 'Acacia', state: 'unset' }])
    expect(getTitles()).toStrictEqual(['Harbour', ...OPENED, 'Acacia'])
  })
})

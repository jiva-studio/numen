import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Tree from './Tree.vue'
import type { Row } from '../lib/row'
import { stubClock } from '@/shared/fixtures/clock'
import type { Clock } from '@/shared/lib/time/clock'

const ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    hasChildren: true,
    rows: [
      {
        id: 'plans',
        name: 'Plans',
        hasChildren: true,
        rows: [{ id: 'friday', name: 'Friday', hasChildren: false }],
      },
      { id: 'notes', name: 'Notes', hasChildren: false },
    ],
  },
  { id: 'empty', name: 'Empty', hasChildren: true },
  { id: 'loose', name: 'Loose', hasChildren: false },
]

/** One row every 24 down the page: work, plans, notes, empty, loose. */
const HEIGHT = 24

/** A clock whose next frame is now. */
const atOnce: Clock = {
  now: () => 0,
  schedule: (run) => {
    run(0)
    return 0
  },
  cancel: vi.fn(),
}

const mountTree = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(Tree, {
    attachTo: document.body,
    props: { rows: ROWS, open: ['work'], clock: atOnce, ...props },
    slots,
  })

const createRect = (top: number, height: number): DOMRect =>
  ({
    x: 0,
    y: top,
    top,
    bottom: top + height,
    left: 0,
    right: 200,
    width: 200,
    height,
    toJSON: () => ({}),
  }) as DOMRect

beforeEach(() => {
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    if (this.hasAttribute('data-tree-row')) {
      const adjacentElement = [...(this.parentElement?.children ?? [])]
      return createRect(adjacentElement.indexOf(this) * HEIGHT, HEIGHT)
    }
    // The rows and the box around them stand as tall as the rows drawn.
    const list = this.getAttribute('role') === 'tree' || this.classList.contains('tree')
    return createRect(0, list ? 5 * HEIGHT : 0)
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

type Tree = ReturnType<typeof mountTree>

const getRow = (wrapper: Tree, row: string) => wrapper.get(`[data-tree-row="${row}"]`)

const getDrawnRows = (wrapper: Tree) =>
  wrapper.findAll('[data-tree-row]').map((row) => row.attributes('data-tree-row'))

/** A pointer event of its own making: the button and the point are read-only. */
const pointer = (
  kind: string,
  y: number,
  on: EventTarget = window,
  overrides: MouseEventInit = {},
): void => {
  on.dispatchEvent(
    new MouseEvent(kind, { bubbles: true, button: 0, clientX: 10, clientY: y, ...overrides }),
  )
}

/** The middle of a row, down the page. */
const getMiddle = (wrapper: Tree, row: string): number => {
  const box = getRow(wrapper, row).element.getBoundingClientRect()
  return box.top + box.height / 2
}

/** A row pressed and let go, with whatever was held down as it was. */
const press = async (wrapper: Tree, row: string, overrides: MouseEventInit = {}): Promise<void> => {
  const on = getRow(wrapper, row)
  const pos = getMiddle(wrapper, row)

  pointer('pointerdown', pos, on.element, overrides)
  pointer('pointerup', pos)
  await on.trigger('click', overrides)
}

/** Drags a row to a vertical position and drops it, in several pointer steps. */
const dragTo = async (wrapper: Tree, row: string, y: number): Promise<void> => {
  const from = getRow(wrapper, row).element
  const start = getMiddle(wrapper, row)

  pointer('pointerdown', start, from)
  pointer('pointermove', (start + y) / 2)
  pointer('pointermove', y)
  pointer('pointerup', y)
  await wrapper.vm.$nextTick()
}

const findDragged = (wrapper: Tree) => wrapper.find('.tree__dragged')

describe('what is drawn', () => {
  it('is the rows an open row holds, in their place', () => {
    expect(getDrawnRows(mountTree())).toStrictEqual(['work', 'plans', 'notes', 'empty', 'loose'])
  })

  it('is nothing a shut row holds', () => {
    const target = mountTree({ open: [] })
    expect(getDrawnRows(target)).toStrictEqual(['work', 'empty', 'loose'])
    expect(target.text()).not.toContain('Plans')
  })

  it('says how deep each row remains, and whether it is open', () => {
    const target = mountTree()
    expect(getRow(target, 'plans').attributes('aria-level')).toBe('2')
    expect(getRow(target, 'work').attributes('aria-expanded')).toBe('true')
    expect(getRow(target, 'plans').attributes('aria-expanded')).toBe('false')
  })

  it('says nothing about opening a row that cannot hold', () => {
    const target = mountTree()
    expect(getRow(target, 'loose').attributes('aria-expanded')).toBeUndefined()
  })

  it('says it holds a selection of several, and which rows are in it', () => {
    const target = mountTree({ selected: ['work', 'notes'] })

    expect(target.get('[role="tree"]').attributes('aria-multiselectable')).toBe('true')
    expect(getRow(target, 'work').attributes('aria-selected')).toBe('true')
    expect(getRow(target, 'notes').attributes('aria-selected')).toBe('true')
    expect(getRow(target, 'plans').attributes('aria-selected')).toBe('false')
  })

  it('offers the keyboard one row of the tree', () => {
    const target = mountTree({ selected: ['notes'] })
    const reachable = target
      .findAll('[data-tree-row]')
      .filter((row) => row.attributes('tabindex') === '0')
    expect(reachable).toHaveLength(1)
    expect(reachable[0]?.attributes('data-tree-row')).toBe('notes')
  })

  it('is what the caller says when there is nothing to draw', () => {
    const target = mountTree({ rows: [] }, { silence: '<i class="mine">No rows</i>' })
    expect(getDrawnRows(target)).toStrictEqual([])
    expect(target.find('.mine').text()).toBe('No rows')
  })
})

describe('a press', () => {
  it('selects the row', async () => {
    const target = mountTree()
    await press(target, 'notes')
    expect(target.emitted('select')).toStrictEqual([[['notes']]])
  })

  it('selects nothing where the row is the selection already', async () => {
    const target = mountTree({ selected: ['notes'] })
    await press(target, 'notes')
    expect(target.emitted('select')).toBeUndefined()
  })

  it('collapses a selection of several onto the row it landed on', async () => {
    const target = mountTree({ selected: ['work', 'notes'] })
    await press(target, 'notes')
    expect(target.emitted('select')).toStrictEqual([[['notes']]])
  })

  it('takes a row into the selection, joined', async () => {
    const target = mountTree({ selected: ['work'] })
    await press(target, 'notes', { ctrlKey: true })
    expect(target.emitted('select')).toStrictEqual([[['work', 'notes']]])
  })

  it('takes a row out of the selection it remains in, joined', async () => {
    const target = mountTree({ selected: ['work', 'notes'] })
    await press(target, 'notes', { metaKey: true })
    expect(target.emitted('select')).toStrictEqual([[['work']]])
  })

  it('reaches from where the last plain press landed', async () => {
    const target = mountTree()
    await press(target, 'work')
    await press(target, 'notes', { shiftKey: true })

    expect(target.emitted('select')?.at(-1)).toStrictEqual([['work', 'plans', 'notes']])
  })

  it('says the selection once for a press with a modifier', async () => {
    const target = mountTree({ selected: ['work'] })
    await press(target, 'notes', { ctrlKey: true })
    expect(target.emitted('select')).toHaveLength(1)
  })

  it('twice on a row that holds turns it', async () => {
    const target = mountTree()
    await getRow(target, 'plans').trigger('dblclick')

    expect(target.emitted('open')).toStrictEqual([['plans']])
  })

  it('twice on an open row shuts it', async () => {
    const target = mountTree()
    await getRow(target, 'work').trigger('dblclick')

    expect(target.emitted('close')).toStrictEqual([['work']])
  })
})

describe('the keyboard', () => {
  const pressKey = (
    wrapper: Tree,
    row: string,
    key: string,
    overrides: Record<string, unknown> = {},
  ) => getRow(wrapper, row).trigger('keydown', { key, ...overrides })

  it('moves the selection a row at a time', async () => {
    const target = mountTree({ selected: ['work'] })
    await pressKey(target, 'work', 'ArrowDown')
    expect(target.emitted('select')).toStrictEqual([[['plans']]])
  })

  it('extends the selection from the anchor with an arrow retained down', async () => {
    const target = mountTree({ selected: ['work'] })
    await press(target, 'work')
    await pressKey(target, 'work', 'ArrowDown', { shiftKey: true })

    expect(target.emitted('select')?.at(-1)).toStrictEqual([['work', 'plans']])
  })

  it('opens a shut row with the right arrow, and moves the selection nowhere', async () => {
    const target = mountTree({ selected: ['plans'] })
    await pressKey(target, 'plans', 'ArrowRight')

    expect(target.emitted('open')).toStrictEqual([['plans']])
    expect(target.emitted('select')).toBeUndefined()
  })

  it('shuts an open row with the left arrow', async () => {
    const target = mountTree({ selected: ['work'] })
    await pressKey(target, 'work', 'ArrowLeft')
    expect(target.emitted('close')).toStrictEqual([['work']])
  })

  it('climbs to the holder from a row that is shut', async () => {
    const target = mountTree({ selected: ['notes'] })
    await pressKey(target, 'notes', 'ArrowLeft')

    expect(target.emitted('select')).toStrictEqual([[['work']]])
    expect(target.emitted('close')).toBeUndefined()
  })

  it('selects every row that is drawn', async () => {
    const target = mountTree()
    await pressKey(target, 'notes', 'a', { ctrlKey: true })

    expect(target.emitted('select')).toStrictEqual([[['work', 'plans', 'notes', 'empty', 'loose']]])
  })

  it('asks for the whole selection to go', async () => {
    const target = mountTree({ selected: ['work', 'notes'] })
    await pressKey(target, 'notes', 'Delete')
    await pressKey(target, 'notes', 'Backspace')

    expect(target.emitted('remove')).toStrictEqual([[['work', 'notes']], [['work', 'notes']]])
  })

  it('asks for nothing to go while nothing is selected', async () => {
    const target = mountTree()
    await pressKey(target, 'notes', 'Delete')
    expect(target.emitted('remove')).toBeUndefined()
  })

  it('puts the row the keyboard relies on into the selection', async () => {
    const target = mountTree({ selected: ['work'] })
    await getRow(target, 'notes').trigger('focus')
    await pressKey(target, 'notes', ' ')

    expect(target.emitted('select')).toStrictEqual([[['work', 'notes']]])
  })

  it('takes that row out again where it already stood in it', async () => {
    const target = mountTree({ selected: ['work', 'notes'] })
    await getRow(target, 'notes').trigger('focus')
    await pressKey(target, 'notes', ' ')

    expect(target.emitted('select')).toStrictEqual([[['work']]])
  })

  it('keeps the space to itself', () => {
    const target = mountTree({ selected: ['work'] })
    const press = new KeyboardEvent('keydown', { key: ' ', bubbles: true, cancelable: true })
    getRow(target, 'notes').element.dispatchEvent(press)

    expect(press.defaultPrevented).toBe(true)
  })

  it('acts on a row that cannot hold, and opens nothing', async () => {
    const target = mountTree({ selected: ['notes'] })
    await pressKey(target, 'notes', 'Enter')

    expect(target.emitted('activate')).toStrictEqual([['notes']])
    expect(target.emitted('open')).toBeUndefined()
  })

  it('turns a row that holds as it acts on it', async () => {
    const target = mountTree({ selected: ['plans'] })
    await pressKey(target, 'plans', 'Enter')

    expect(target.emitted('activate')).toStrictEqual([['plans']])
    expect(target.emitted('open')).toStrictEqual([['plans']])
  })

  it('reaches the menu the pointer reaches', async () => {
    const target = mountTree({ selected: ['notes'] })
    await pressKey(target, 'notes', 'F10', { shiftKey: true })

    expect(target.emitted('menu')).toStrictEqual([['notes', { x: 0, y: 3 * HEIGHT }]])
  })
})

describe('a menu requested for', () => {
  it('is requested for on the row, and the selection relies on it', async () => {
    const target = mountTree()
    await getRow(target, 'notes').trigger('contextmenu', { clientX: 40, clientY: 60 })

    expect(target.emitted('select')).toStrictEqual([[['notes']]])
    expect(target.emitted('menu')).toStrictEqual([['notes', { x: 40, y: 60 }]])
  })

  it('leaves a selection of several alone where the row remains in it', async () => {
    const target = mountTree({ selected: ['work', 'notes'] })
    await getRow(target, 'notes').trigger('contextmenu', { clientX: 40, clientY: 60 })

    expect(target.emitted('select')).toBeUndefined()
  })

  it('names no row at all, requested off every row', async () => {
    const target = mountTree()
    await target.get('.tree').trigger('contextmenu', { clientX: 4, clientY: 8 })

    expect(target.emitted('menu')).toStrictEqual([[null, { x: 4, y: 8 }]])
  })

  it('names no row at all where there is nothing to draw', async () => {
    const target = mountTree({ rows: [] })
    await target.get('.tree').trigger('contextmenu', { clientX: 4, clientY: 8 })

    expect(target.emitted('menu')).toStrictEqual([[null, { x: 4, y: 8 }]])
  })
})

describe('a drag', () => {
  it('moves a row into one that holds', async () => {
    const target = mountTree()
    await dragTo(target, 'loose', 12)
    expect(target.emitted('move')).toStrictEqual([[['loose'], { into: 'work' }]])
  })

  it('moves a row between two others', async () => {
    const target = mountTree()
    await dragTo(target, 'loose', 2)
    expect(target.emitted('move')).toStrictEqual([[['loose'], { before: 'work' }]])
  })

  it('drags the whole selection, off a row standing in it', async () => {
    const target = mountTree({ selected: ['notes', 'loose'] })
    await dragTo(target, 'loose', 12)

    expect(target.emitted('move')).toStrictEqual([[['notes', 'loose'], { into: 'work' }]])
    expect(target.emitted('select')).toBeUndefined()
  })

  it('selects a row standing outside the selection, and drags it alone', async () => {
    const target = mountTree({ selected: ['notes'] })
    await dragTo(target, 'loose', 12)

    expect(target.emitted('select')).toStrictEqual([[['loose']]])
    expect(target.emitted('move')).toStrictEqual([[['loose'], { into: 'work' }]])
  })

  it('is rejected into what the row holds, and does not move', async () => {
    const target = mountTree()
    await dragTo(target, 'work', HEIGHT + 12)
    expect(target.emitted('move')).toBeUndefined()
  })

  it('is rejected into what any of the rows dragged holds, and does not move', async () => {
    const target = mountTree({ selected: ['work', 'loose'] })
    await dragTo(target, 'loose', HEIGHT + 12)
    expect(target.emitted('move')).toBeUndefined()
  })

  it('takes a row to the top level, let go past the last row', async () => {
    const target = mountTree()
    await dragTo(target, 'work', 5 * HEIGHT - 2)
    expect(target.emitted('move')).toStrictEqual([[['work'], { into: null }]])
  })

  // The pointer is caught on the window, so a row let go over another pane is
  // let go somewhere the tree does not answer for.
  it('does not move let go alongside the tree', async () => {
    const target = mountTree()
    const from = getRow(target, 'work').element

    pointer('pointerdown', getMiddle(target, 'work'), from)
    pointer('pointermove', 2 * HEIGHT, window, { clientX: 400 })
    pointer('pointerup', 2 * HEIGHT, window, { clientX: 400 })
    await target.vm.$nextTick()

    expect(target.emitted('move')).toBeUndefined()
  })

  it('does not move let go below the tree', async () => {
    const target = mountTree()
    await dragTo(target, 'work', 9 * HEIGHT)

    expect(target.emitted('move')).toBeUndefined()
  })

  it('does not move where the pointer did not travel far enough', async () => {
    const target = mountTree()
    pointer('pointerdown', 108, getRow(target, 'loose').element)
    pointer('pointermove', 110)
    pointer('pointerup', 110)
    await target.vm.$nextTick()

    expect(target.emitted('move')).toBeUndefined()
  })

  it('leaves the press that follows it standing down', async () => {
    const world = stubClock()
    const target = mountTree({ selected: ['loose'], clock: world.clock })

    await dragTo(target, 'loose', 12)
    await getRow(target, 'notes').trigger('click')

    expect(target.emitted('select')).toBeUndefined()

    world.run()
    await getRow(target, 'notes').trigger('click')
    expect(target.emitted('select')).toStrictEqual([[['notes']]])
  })
})

describe('rows dragged out of the tree', () => {
  it('says which rows are being dragged, once for the whole drag', async () => {
    const target = mountTree({ selected: ['notes', 'loose'] })
    const from = getRow(target, 'loose').element

    pointer('pointerdown', getMiddle(target, 'loose'), from)
    pointer('pointermove', 3 * HEIGHT)
    pointer('pointermove', 4 * HEIGHT)
    await target.vm.$nextTick()

    expect(target.emitted('drag')).toStrictEqual([[['notes', 'loose']]])
  })

  it('says so for a drag over another pane, where nothing in the tree is landed on', async () => {
    const target = mountTree()
    const from = getRow(target, 'work').element

    pointer('pointerdown', getMiddle(target, 'work'), from)
    pointer('pointermove', 2 * HEIGHT, window, { clientX: 400 })
    pointer('pointerup', 2 * HEIGHT, window, { clientX: 400 })
    await target.vm.$nextTick()

    expect(target.emitted('drag')).toStrictEqual([[['work']]])
    expect(target.emitted('drop')).toStrictEqual([[]])
    expect(target.emitted('move')).toBeUndefined()
  })

  it('says it has let go after saying where the rows landed', async () => {
    const target = mountTree()
    await dragTo(target, 'loose', 12)

    expect(target.emitted('move')).toStrictEqual([[['loose'], { into: 'work' }]])
    expect(target.emitted('drop')).toStrictEqual([[]])
  })

  it('drags nothing where the pointer did not travel far enough', async () => {
    const target = mountTree()
    pointer('pointerdown', 108, getRow(target, 'loose').element)
    pointer('pointermove', 110)
    pointer('pointerup', 110)
    await target.vm.$nextTick()

    expect(target.emitted('drag')).toBeUndefined()
    expect(target.emitted('drop')).toBeUndefined()
  })
})

describe('what follows the pointer', () => {
  it('says the name of the one row being dragged', async () => {
    const target = mountTree()
    pointer('pointerdown', 108, getRow(target, 'loose').element)
    pointer('pointermove', 60)
    await target.vm.$nextTick()

    expect(findDragged(target).text()).toBe('Loose')
  })

  it('says how many are being dragged, where there are several', async () => {
    const target = mountTree({ selected: ['notes', 'loose'] })
    pointer('pointerdown', 108, getRow(target, 'loose').element)
    pointer('pointermove', 60)
    await target.vm.$nextTick()

    expect(findDragged(target).text()).toBe('2 rows')
  })

  it('says it in the words the caller gave for how many', async () => {
    const wrapper = mountTree({
      selected: ['notes', 'loose'],
      counted: (rows: number) => `${rows} files`,
    })
    pointer('pointerdown', 108, getRow(wrapper, 'loose').element)
    pointer('pointermove', 60)
    await wrapper.vm.$nextTick()

    expect(findDragged(wrapper).text()).toBe('2 files')
  })

  it('remains where the pointer is', async () => {
    const wrapper = mountTree()
    pointer('pointerdown', 108, getRow(wrapper, 'loose').element)
    pointer('pointermove', 60)
    await wrapper.vm.$nextTick()

    expect(findDragged(wrapper).attributes('style')).toContain('top: 60px')
  })

  it('is drawn nowhere before the pointer has travelled far enough', async () => {
    const wrapper = mountTree()
    pointer('pointerdown', 108, getRow(wrapper, 'loose').element)
    pointer('pointermove', 110)
    await wrapper.vm.$nextTick()

    expect(findDragged(wrapper).exists()).toBe(false)
  })

  it('is drawn nowhere once the rows have been release', async () => {
    const wrapper = mountTree()
    await dragTo(wrapper, 'loose', 12)

    expect(findDragged(wrapper).exists()).toBe(false)
  })

  it('is drawn nowhere at all while nothing is being dragged', () => {
    expect(findDragged(mountTree()).exists()).toBe(false)
  })
})

describe('a name being typed', () => {
  const getField = (wrapper: Tree) => wrapper.get('.tree__field')

  it('remains over the row it belongs to and over no other', () => {
    const wrapper = mountTree({ renamingPath: 'notes' })

    expect(getRow(wrapper, 'notes').find('.tree__field').exists()).toBe(true)
    expect(wrapper.findAll('.tree__field')).toHaveLength(1)
    expect(getRow(wrapper, 'notes').find('.tree__name').exists()).toBe(false)
  })

  it('starts as the name the row already carries', () => {
    expect((getField(mountTree({ renamingPath: 'notes' })).element as HTMLInputElement).value).toBe(
      'Notes',
    )
  })

  it('is committed on Enter', async () => {
    const wrapper = mountTree({ renamingPath: 'notes' })
    await getField(wrapper).setValue('Friday notes')
    await getField(wrapper).trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('rename')).toStrictEqual([['notes', 'Friday notes']])
    expect(wrapper.emitted('update:renamingPath')).toStrictEqual([[null]])
  })

  it('is abandoned on Escape, and does not rename', async () => {
    const wrapper = mountTree({ renamingPath: 'notes' })
    await getField(wrapper).setValue('Friday notes')
    await getField(wrapper).trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('rename')).toBeUndefined()
    expect(wrapper.emitted('update:renamingPath')).toStrictEqual([[null]])
  })

  it('keeps the arrows to itself while it is being typed in', async () => {
    const wrapper = mountTree({ renamingPath: 'notes', selected: ['notes'] })
    await getField(wrapper).trigger('keydown', { key: 'ArrowDown' })

    expect(wrapper.emitted('select')).toBeUndefined()
  })
})

describe('the attribute rows are marked with', () => {
  const ATTRIBUTE = 'data-somewhere'

  const getMark = (wrapper: Tree, row: string) => getRow(wrapper, row).attributes(ATTRIBUTE)

  const getAttributeValue = (row: string | null): string => {
    if (row === null) return ''
    if (row === 'work' || row === 'plans' || row === 'empty') return row
    return row === 'friday' ? 'plans' : 'work'
  }

  it('is written on each row, and on the tree for the top level', () => {
    const wrapper = mountTree({ marking: { attribute: ATTRIBUTE, getAttributeValue } })

    expect(getMark(wrapper, 'plans')).toBe('plans')
    expect(getMark(wrapper, 'notes')).toBe('work')
    expect(wrapper.get('.tree').attributes(ATTRIBUTE)).toBe('')
  })

  it('is written nowhere while the tree is given no marking', () => {
    const wrapper = mountTree()

    expect(getMark(wrapper, 'plans')).toBeUndefined()
    expect(wrapper.get('.tree').attributes(ATTRIBUTE)).toBeUndefined()
  })

  it('is written under the name it was given', () => {
    const wrapper = mountTree({ marking: { attribute: 'data-elsewhere', getAttributeValue } })

    expect(getRow(wrapper, 'plans').attributes('data-elsewhere')).toBe('plans')
    expect(getMark(wrapper, 'plans')).toBeUndefined()
  })

  it('is written on the rows answered for and on no other', () => {
    const marking = {
      attribute: ATTRIBUTE,
      getAttributeValue: (row: string | null) => (row === 'empty' ? 'empty' : null),
    }
    const wrapper = mountTree({ marking })

    expect(getMark(wrapper, 'empty')).toBe('empty')
    expect(getMark(wrapper, 'plans')).toBeUndefined()
  })
})

/** A cancelable key press: the tree took it where it is prevented. */
const sendKey = (
  target: Element,
  key: string,
  overrides: KeyboardEventInit = {},
): KeyboardEvent => {
  const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...overrides })
  target.dispatchEvent(event)
  return event
}

describe('the keys the tree takes', () => {
  const sendKeyOn = (wrapper: Tree, row: string, key: string, overrides: KeyboardEventInit = {}) =>
    sendKey(getRow(wrapper, row).element, key, overrides)

  it('takes Ctrl and A, in either case, to select every row', () => {
    const wrapper = mountTree()
    const lower = sendKeyOn(wrapper, 'notes', 'a', { ctrlKey: true })
    const upper = sendKeyOn(wrapper, 'notes', 'A', { ctrlKey: true })

    expect(lower.defaultPrevented).toBe(true)
    expect(upper.defaultPrevented).toBe(true)
    expect(wrapper.emitted('select')?.[0]).toStrictEqual([
      ['work', 'plans', 'notes', 'empty', 'loose'],
    ])
  })

  it('takes Meta and A to select every row', () => {
    const wrapper = mountTree()
    const acquiredItem = sendKeyOn(wrapper, 'notes', 'a', { metaKey: true })

    expect(acquiredItem.defaultPrevented).toBe(true)
    expect(wrapper.emitted('select')).toStrictEqual([
      [['work', 'plans', 'notes', 'empty', 'loose']],
    ])
  })

  it('leaves A alone while nothing is retained down', () => {
    const wrapper = mountTree()
    const keyResult = sendKeyOn(wrapper, 'notes', 'a')

    expect(keyResult.defaultPrevented).toBe(false)
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('leaves another key alone with Ctrl retained down', () => {
    const wrapper = mountTree()
    const keyResult = sendKeyOn(wrapper, 'notes', 'b', { ctrlKey: true })

    expect(keyResult.defaultPrevented).toBe(false)
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('takes Delete and Backspace', () => {
    const wrapper = mountTree({ selected: ['notes'] })

    expect(sendKeyOn(wrapper, 'notes', 'Delete').defaultPrevented).toBe(true)
    expect(sendKeyOn(wrapper, 'notes', 'Backspace').defaultPrevented).toBe(true)
  })

  it('takes Delete where nothing is selected', () => {
    const wrapper = mountTree()
    expect(sendKeyOn(wrapper, 'notes', 'Delete').defaultPrevented).toBe(true)
  })

  it('takes Enter', () => {
    const wrapper = mountTree({ selected: ['notes'] })
    expect(sendKeyOn(wrapper, 'notes', 'Enter').defaultPrevented).toBe(true)
  })

  it('takes the menu key, and reaches the menu the pointer reaches', () => {
    const wrapper = mountTree({ selected: ['notes'] })
    const acquiredItem = sendKeyOn(wrapper, 'notes', 'ContextMenu')

    expect(acquiredItem.defaultPrevented).toBe(true)
    expect(wrapper.emitted('menu')).toStrictEqual([['notes', { x: 0, y: 3 * HEIGHT }]])
  })

  it('takes F10 only with Shift', () => {
    const wrapper = mountTree({ selected: ['notes'] })
    const keyResult = sendKeyOn(wrapper, 'notes', 'F10')

    expect(keyResult.defaultPrevented).toBe(false)
    expect(wrapper.emitted('menu')).toBeUndefined()
  })

  it('takes the arrows, the Home key and the End key', () => {
    const wrapper = mountTree({ selected: ['notes'] })
    const keys = ['ArrowDown', 'ArrowUp', 'ArrowRight', 'ArrowLeft', 'Home', 'End']

    expect(keys.map((key) => sendKeyOn(wrapper, 'notes', key).defaultPrevented)).toStrictEqual(
      keys.map(() => true),
    )
  })

  it('leaves any other key alone', () => {
    const wrapper = mountTree({ selected: ['notes'] })
    const keyResult = sendKeyOn(wrapper, 'notes', 'x')

    expect(keyResult.defaultPrevented).toBe(false)
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('leaves the arrows alone while there is nothing to draw', () => {
    const wrapper = mountTree({ rows: [] })
    const keyResult = sendKey(wrapper.get('[role="tree"]').element, 'ArrowDown')

    expect(keyResult.defaultPrevented).toBe(false)
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('leaves Enter alone while there is nothing to draw', () => {
    const wrapper = mountTree({ rows: [] })
    const keyResult = sendKey(wrapper.get('[role="tree"]').element, 'Enter')

    expect(keyResult.defaultPrevented).toBe(false)
    expect(wrapper.emitted('activate')).toBeUndefined()
  })
})

describe('a press on a row', () => {
  /** A press of its own making: it can be cancelled, and the button is the one given. */
  const sendPress = (wrapper: Tree, row: string, button: number): MouseEvent => {
    const event = new MouseEvent('pointerdown', {
      bubbles: true,
      cancelable: true,
      button,
      clientX: 10,
      clientY: getMiddle(wrapper, row),
    })
    getRow(wrapper, row).element.dispatchEvent(event)
    return event
  }

  it('is taken with the primary button, and puts the keyboard on the row', () => {
    const wrapper = mountTree()
    const acquiredItem = sendPress(wrapper, 'notes', 0)

    expect(acquiredItem.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(getRow(wrapper, 'notes').element)
    expect(wrapper.emitted('select')).toStrictEqual([[['notes']]])
  })

  it('is left alone with any other button', () => {
    const wrapper = mountTree()
    const keyResult = sendPress(wrapper, 'notes', 2)

    expect(keyResult.defaultPrevented).toBe(false)
    expect(document.activeElement).not.toBe(getRow(wrapper, 'notes').element)
    expect(wrapper.emitted('select')).toBeUndefined()
  })
})

describe('where a drop would land, shown on the tree', () => {
  const lift = (wrapper: Tree, row: string, y: number): void => {
    pointer('pointerdown', getMiddle(wrapper, row), getRow(wrapper, row).element)
    pointer('pointermove', y)
  }

  const marked = (wrapper: Tree, attribute: string) =>
    wrapper.findAll(`[data-tree-row][${attribute}]`).map((row) => row.attributes('data-tree-row'))

  it('marks the row a drop would land inside, and no row above', async () => {
    const wrapper = mountTree()
    lift(wrapper, 'loose', 12)
    await wrapper.vm.$nextTick()

    expect(marked(wrapper, 'data-into')).toStrictEqual(['work'])
    expect(marked(wrapper, 'data-before')).toStrictEqual([])
    expect(wrapper.get('.tree').attributes('data-into')).toBeUndefined()
  })

  it('marks the row a drop would land above, and no row inside', async () => {
    const wrapper = mountTree()
    lift(wrapper, 'loose', 2)
    await wrapper.vm.$nextTick()

    expect(marked(wrapper, 'data-before')).toStrictEqual(['work'])
    expect(marked(wrapper, 'data-into')).toStrictEqual([])
  })

  it('marks the whole tree where a drop would land at the top level', async () => {
    const wrapper = mountTree()
    lift(wrapper, 'work', 5 * HEIGHT - 2)
    await wrapper.vm.$nextTick()

    expect(wrapper.get('.tree').attributes('data-into')).toBe('')
    expect(marked(wrapper, 'data-into')).toStrictEqual([])
    expect(marked(wrapper, 'data-before')).toStrictEqual([])
  })

  it('marks no row where a drop would land nowhere', async () => {
    const wrapper = mountTree()
    lift(wrapper, 'loose', 9 * HEIGHT)
    await wrapper.vm.$nextTick()

    expect(marked(wrapper, 'data-into')).toStrictEqual([])
    expect(marked(wrapper, 'data-before')).toStrictEqual([])
  })

  it('marks the rows being dragged while they are, and no other', async () => {
    const wrapper = mountTree({ selected: ['notes', 'loose'] })
    lift(wrapper, 'loose', 12)
    await wrapper.vm.$nextTick()

    expect(marked(wrapper, 'data-dragged')).toStrictEqual(['notes', 'loose'])
  })

  it('marks no row once the rows have been release', async () => {
    const wrapper = mountTree({ selected: ['notes', 'loose'] })
    await dragTo(wrapper, 'loose', 12)

    expect(marked(wrapper, 'data-dragged')).toStrictEqual([])
    expect(marked(wrapper, 'data-into')).toStrictEqual([])
  })
})

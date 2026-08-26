/**
 * What the tree draws from the rows it was handed, and what it emits.
 *
 * The negatives are here: a shut row's contents are not drawn, a refused drop
 * moves nothing, and a name abandoned renames nothing. A drag is read from
 * heights, and the rows are given theirs by hand.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Tree from './Tree.vue'
import type { Row } from './model'

const ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    holds: true,
    rows: [
      {
        id: 'plans',
        name: 'Plans',
        holds: true,
        rows: [{ id: 'friday', name: 'Friday', holds: false }],
      },
      { id: 'notes', name: 'Notes', holds: false },
    ],
  },
  { id: 'empty', name: 'Empty', holds: true },
  { id: 'loose', name: 'Loose', holds: false },
]

/** One row every 24 down the page: work, plans, notes, empty, loose. */
const HEIGHT = 24

const mountTree = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(Tree, {
    attachTo: document.body,
    props: { rows: ROWS, open: ['work'], frame: (run: () => void) => run(), ...props },
    slots,
  })

const boxOf = (top: number, height: number): DOMRect =>
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
      const beside = [...(this.parentElement?.children ?? [])]
      return boxOf(beside.indexOf(this) * HEIGHT, HEIGHT)
    }
    return boxOf(0, this.getAttribute('role') === 'tree' ? 5 * HEIGHT : 0)
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

type Tree = ReturnType<typeof mountTree>

const rowIn = (held: Tree, row: string) => held.get(`[data-tree-row="${row}"]`)

const drawn = (held: Tree) =>
  held.findAll('[data-tree-row]').map((row) => row.attributes('data-tree-row'))

/** A pointer event of its own making: the button and the point are read-only. */
const pointer = (
  kind: string,
  y: number,
  on: EventTarget = window,
  over: MouseEventInit = {},
): void => {
  on.dispatchEvent(
    new MouseEvent(kind, { bubbles: true, button: 0, clientX: 10, clientY: y, ...over }),
  )
}

/** The middle of a row, down the page. */
const middleOf = (held: Tree, row: string): number => {
  const box = rowIn(held, row).element.getBoundingClientRect()
  return box.top + box.height / 2
}

/** A row pressed and let go, with whatever was held down as it was. */
const press = async (held: Tree, row: string, over: MouseEventInit = {}): Promise<void> => {
  const on = rowIn(held, row)
  const at = middleOf(held, row)

  pointer('pointerdown', at, on.element, over)
  pointer('pointerup', at)
  await on.trigger('click', over)
}

/** A row picked up and let go at a height, in as many steps as a hand takes. */
const dragTo = async (held: Tree, row: string, y: number): Promise<void> => {
  const from = rowIn(held, row).element
  const at = middleOf(held, row)

  pointer('pointerdown', at, from)
  pointer('pointermove', (at + y) / 2)
  pointer('pointermove', y)
  pointer('pointerup', y)
  await held.vm.$nextTick()
}

const carriedIn = (held: Tree) => held.find('.tree__carried')

describe('what is drawn', () => {
  it('is the rows an open row holds, in their place', () => {
    expect(drawn(mountTree())).toStrictEqual(['work', 'plans', 'notes', 'empty', 'loose'])
  })

  it('is nothing a shut row holds', () => {
    const held = mountTree({ open: [] })
    expect(drawn(held)).toStrictEqual(['work', 'empty', 'loose'])
    expect(held.text()).not.toContain('Plans')
  })

  it('says how deep each row stands, and whether it is open', () => {
    const held = mountTree()
    expect(rowIn(held, 'plans').attributes('aria-level')).toBe('2')
    expect(rowIn(held, 'work').attributes('aria-expanded')).toBe('true')
    expect(rowIn(held, 'plans').attributes('aria-expanded')).toBe('false')
  })

  it('says nothing about opening a row that cannot hold', () => {
    const held = mountTree()
    expect(rowIn(held, 'loose').attributes('aria-expanded')).toBeUndefined()
    expect(rowIn(held, 'loose').find('[data-tree-twist]').exists()).toBe(false)
  })

  it('says it holds a selection of several, and which rows are in it', () => {
    const held = mountTree({ selected: ['work', 'notes'] })

    expect(held.get('[role="tree"]').attributes('aria-multiselectable')).toBe('true')
    expect(rowIn(held, 'work').attributes('aria-selected')).toBe('true')
    expect(rowIn(held, 'notes').attributes('aria-selected')).toBe('true')
    expect(rowIn(held, 'plans').attributes('aria-selected')).toBe('false')
  })

  it('offers the keyboard one row of the tree', () => {
    const held = mountTree({ selected: ['notes'] })
    const reachable = held
      .findAll('[data-tree-row]')
      .filter((row) => row.attributes('tabindex') === '0')
    expect(reachable).toHaveLength(1)
    expect(reachable[0]?.attributes('data-tree-row')).toBe('notes')
  })

  it('is what the caller says when there is nothing to draw', () => {
    const held = mountTree({ rows: [] }, { silence: '<i class="mine">No rows</i>' })
    expect(drawn(held)).toStrictEqual([])
    expect(held.find('.mine').text()).toBe('No rows')
  })
})

describe('a press', () => {
  it('selects the row', async () => {
    const held = mountTree()
    await press(held, 'notes')
    expect(held.emitted('select')).toStrictEqual([[['notes']]])
  })

  it('selects nothing where the row is the selection already', async () => {
    const held = mountTree({ selected: ['notes'] })
    await press(held, 'notes')
    expect(held.emitted('select')).toBeUndefined()
  })

  it('collapses a selection of several onto the row it landed on', async () => {
    const held = mountTree({ selected: ['work', 'notes'] })
    await press(held, 'notes')
    expect(held.emitted('select')).toStrictEqual([[['notes']]])
  })

  it('takes a row into the selection, joined', async () => {
    const held = mountTree({ selected: ['work'] })
    await press(held, 'notes', { ctrlKey: true })
    expect(held.emitted('select')).toStrictEqual([[['work', 'notes']]])
  })

  it('takes a row out of the selection it stands in, joined', async () => {
    const held = mountTree({ selected: ['work', 'notes'] })
    await press(held, 'notes', { metaKey: true })
    expect(held.emitted('select')).toStrictEqual([[['work']]])
  })

  it('reaches from where the last plain press landed', async () => {
    const held = mountTree()
    await press(held, 'work')
    await press(held, 'notes', { shiftKey: true })

    expect(held.emitted('select')?.at(-1)).toStrictEqual([['work', 'plans', 'notes']])
  })

  it('says the selection once for a press with a modifier', async () => {
    const held = mountTree({ selected: ['work'] })
    await press(held, 'notes', { ctrlKey: true })
    expect(held.emitted('select')).toHaveLength(1)
  })

  it('on the disclosure turns the row and selects nothing', async () => {
    const held = mountTree()
    await rowIn(held, 'plans').get('[data-tree-twist]').trigger('click')

    expect(held.emitted('open')).toStrictEqual([['plans']])
    expect(held.emitted('select')).toBeUndefined()
  })

  it('on an open disclosure shuts the row', async () => {
    const held = mountTree()
    await rowIn(held, 'work').get('[data-tree-twist]').trigger('click')
    expect(held.emitted('close')).toStrictEqual([['work']])
  })
})

describe('the keyboard', () => {
  const types = (held: Tree, row: string, key: string, over: Record<string, unknown> = {}) =>
    rowIn(held, row).trigger('keydown', { key, ...over })

  it('moves the selection a row at a time', async () => {
    const held = mountTree({ selected: ['work'] })
    await types(held, 'work', 'ArrowDown')
    expect(held.emitted('select')).toStrictEqual([[['plans']]])
  })

  it('extends the selection from the anchor with an arrow held down', async () => {
    const held = mountTree({ selected: ['work'] })
    await press(held, 'work')
    await types(held, 'work', 'ArrowDown', { shiftKey: true })

    expect(held.emitted('select')?.at(-1)).toStrictEqual([['work', 'plans']])
  })

  it('opens a shut row with the right arrow, and moves the selection nowhere', async () => {
    const held = mountTree({ selected: ['plans'] })
    await types(held, 'plans', 'ArrowRight')

    expect(held.emitted('open')).toStrictEqual([['plans']])
    expect(held.emitted('select')).toBeUndefined()
  })

  it('shuts an open row with the left arrow', async () => {
    const held = mountTree({ selected: ['work'] })
    await types(held, 'work', 'ArrowLeft')
    expect(held.emitted('close')).toStrictEqual([['work']])
  })

  it('climbs to the holder from a row that is shut', async () => {
    const held = mountTree({ selected: ['notes'] })
    await types(held, 'notes', 'ArrowLeft')

    expect(held.emitted('select')).toStrictEqual([[['work']]])
    expect(held.emitted('close')).toBeUndefined()
  })

  it('selects every row that is drawn', async () => {
    const held = mountTree()
    await types(held, 'notes', 'a', { ctrlKey: true })

    expect(held.emitted('select')).toStrictEqual([
      [['work', 'plans', 'notes', 'empty', 'loose']],
    ])
  })

  it('asks for the whole selection to go', async () => {
    const held = mountTree({ selected: ['work', 'notes'] })
    await types(held, 'notes', 'Delete')
    await types(held, 'notes', 'Backspace')

    expect(held.emitted('remove')).toStrictEqual([[['work', 'notes']], [['work', 'notes']]])
  })

  it('asks for nothing to go while nothing is selected', async () => {
    const held = mountTree()
    await types(held, 'notes', 'Delete')
    expect(held.emitted('remove')).toBeUndefined()
  })

  it('acts on a row that cannot hold, and opens nothing', async () => {
    const held = mountTree({ selected: ['notes'] })
    await types(held, 'notes', 'Enter')

    expect(held.emitted('activate')).toStrictEqual([['notes']])
    expect(held.emitted('open')).toBeUndefined()
  })

  it('turns a row that holds as it acts on it', async () => {
    const held = mountTree({ selected: ['plans'] })
    await types(held, 'plans', 'Enter')

    expect(held.emitted('activate')).toStrictEqual([['plans']])
    expect(held.emitted('open')).toStrictEqual([['plans']])
  })

  it('reaches the menu the pointer reaches', async () => {
    const held = mountTree({ selected: ['notes'] })
    await types(held, 'notes', 'F10', { shiftKey: true })

    expect(held.emitted('menu')).toStrictEqual([['notes', { x: 0, y: 3 * HEIGHT }]])
  })
})

describe('a menu asked for', () => {
  it('is asked for on the row, and the selection stands on it', async () => {
    const held = mountTree()
    await rowIn(held, 'notes').trigger('contextmenu', { clientX: 40, clientY: 60 })

    expect(held.emitted('select')).toStrictEqual([[['notes']]])
    expect(held.emitted('menu')).toStrictEqual([['notes', { x: 40, y: 60 }]])
  })

  it('leaves a selection of several alone where the row stands in it', async () => {
    const held = mountTree({ selected: ['work', 'notes'] })
    await rowIn(held, 'notes').trigger('contextmenu', { clientX: 40, clientY: 60 })

    expect(held.emitted('select')).toBeUndefined()
  })

  it('names no row at all, asked off every row', async () => {
    const held = mountTree()
    await held.get('.tree').trigger('contextmenu', { clientX: 4, clientY: 8 })

    expect(held.emitted('menu')).toStrictEqual([[null, { x: 4, y: 8 }]])
  })

  it('names no row at all where there is nothing to draw', async () => {
    const held = mountTree({ rows: [] })
    await held.get('.tree').trigger('contextmenu', { clientX: 4, clientY: 8 })

    expect(held.emitted('menu')).toStrictEqual([[null, { x: 4, y: 8 }]])
  })
})

describe('a drag', () => {
  it('moves a row into one that holds', async () => {
    const held = mountTree()
    await dragTo(held, 'loose', 12)
    expect(held.emitted('move')).toStrictEqual([[['loose'], { into: 'work' }]])
  })

  it('moves a row between two others', async () => {
    const held = mountTree()
    await dragTo(held, 'loose', 2)
    expect(held.emitted('move')).toStrictEqual([[['loose'], { before: 'work' }]])
  })

  it('carries the whole selection, off a row standing in it', async () => {
    const held = mountTree({ selected: ['notes', 'loose'] })
    await dragTo(held, 'loose', 12)

    expect(held.emitted('move')).toStrictEqual([[['notes', 'loose'], { into: 'work' }]])
    expect(held.emitted('select')).toBeUndefined()
  })

  it('selects a row standing outside the selection, and carries it alone', async () => {
    const held = mountTree({ selected: ['notes'] })
    await dragTo(held, 'loose', 12)

    expect(held.emitted('select')).toStrictEqual([[['loose']]])
    expect(held.emitted('move')).toStrictEqual([[['loose'], { into: 'work' }]])
  })

  it('is refused into what the row holds, and moves nothing', async () => {
    const held = mountTree()
    await dragTo(held, 'work', HEIGHT + 12)
    expect(held.emitted('move')).toBeUndefined()
  })

  it('is refused into what any of the rows carried holds, and moves nothing', async () => {
    const held = mountTree({ selected: ['work', 'loose'] })
    await dragTo(held, 'loose', HEIGHT + 12)
    expect(held.emitted('move')).toBeUndefined()
  })

  it('moves nothing let go past the last row', async () => {
    const held = mountTree()
    await dragTo(held, 'work', 5 * HEIGHT - 2)
    expect(held.emitted('move')).toBeUndefined()
  })

  it('moves nothing where the pointer did not travel far enough', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 110)
    pointer('pointerup', 110)
    await held.vm.$nextTick()

    expect(held.emitted('move')).toBeUndefined()
  })

  it('leaves the press that follows it standing down', async () => {
    const frames: (() => void)[] = []
    const held = mountTree({
      selected: ['loose'],
      frame: (run: () => void) => frames.push(run),
    })

    await dragTo(held, 'loose', 12)
    await rowIn(held, 'notes').trigger('click')

    expect(held.emitted('select')).toBeUndefined()

    frames.forEach((run) => run())
    await rowIn(held, 'notes').trigger('click')
    expect(held.emitted('select')).toStrictEqual([[['notes']]])
  })
})

describe('what follows the pointer', () => {
  it('says the name of the one row being carried', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(carriedIn(held).text()).toBe('Loose')
  })

  it('says how many are being carried, where there are several', async () => {
    const held = mountTree({ selected: ['notes', 'loose'] })
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(carriedIn(held).text()).toBe('2 rows')
  })

  it('says it in the words the caller gave for how many', async () => {
    const held = mountTree({
      selected: ['notes', 'loose'],
      counted: (rows: number) => `${rows} files`,
    })
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(carriedIn(held).text()).toBe('2 files')
  })

  it('stands where the pointer is', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(carriedIn(held).attributes('style')).toContain('top: 60px')
  })

  it('is drawn nowhere before the pointer has travelled far enough', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 110)
    await held.vm.$nextTick()

    expect(carriedIn(held).exists()).toBe(false)
  })

  it('is drawn nowhere once the rows have been let go of', async () => {
    const held = mountTree()
    await dragTo(held, 'loose', 12)

    expect(carriedIn(held).exists()).toBe(false)
  })

  it('is drawn nowhere at all while nothing is being carried', () => {
    expect(carriedIn(mountTree()).exists()).toBe(false)
  })
})

describe('a name being typed', () => {
  const fieldIn = (held: Tree) => held.get('.tree__field')

  it('stands over the row it belongs to and over no other', () => {
    const held = mountTree({ renaming: 'notes' })

    expect(rowIn(held, 'notes').find('.tree__field').exists()).toBe(true)
    expect(held.findAll('.tree__field')).toHaveLength(1)
    expect(rowIn(held, 'notes').find('.tree__name').exists()).toBe(false)
  })

  it('starts as the name the row already carries', () => {
    expect((fieldIn(mountTree({ renaming: 'notes' })).element as HTMLInputElement).value).toBe(
      'Notes',
    )
  })

  it('is committed on Enter', async () => {
    const held = mountTree({ renaming: 'notes' })
    await fieldIn(held).setValue('Friday notes')
    await fieldIn(held).trigger('keydown', { key: 'Enter' })

    expect(held.emitted('rename')).toStrictEqual([['notes', 'Friday notes']])
    expect(held.emitted('update:renaming')).toStrictEqual([[null]])
  })

  it('is abandoned on Escape, and renames nothing', async () => {
    const held = mountTree({ renaming: 'notes' })
    await fieldIn(held).setValue('Friday notes')
    await fieldIn(held).trigger('keydown', { key: 'Escape' })

    expect(held.emitted('rename')).toBeUndefined()
    expect(held.emitted('update:renaming')).toStrictEqual([[null]])
  })

  it('keeps the arrows to itself while it is being typed in', async () => {
    const held = mountTree({ renaming: 'notes', selected: ['notes'] })
    await fieldIn(held).trigger('keydown', { key: 'ArrowDown' })

    expect(held.emitted('select')).toBeUndefined()
  })
})

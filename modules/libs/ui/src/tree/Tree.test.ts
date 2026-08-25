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
const pointer = (kind: string, y: number, on: EventTarget = window): void => {
  on.dispatchEvent(new MouseEvent(kind, { bubbles: true, button: 0, clientX: 10, clientY: y }))
}

/** A row picked up and let go at a height, in as many steps as a hand takes. */
const dragTo = async (held: Tree, row: string, y: number): Promise<void> => {
  const from = rowIn(held, row).element
  const box = from.getBoundingClientRect()
  const at = box.top + box.height / 2

  pointer('pointerdown', at, from)
  pointer('pointermove', (at + y) / 2)
  pointer('pointermove', y)
  pointer('pointerup', y)
  await held.vm.$nextTick()
}

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

  it('offers the keyboard one row of the tree', () => {
    const held = mountTree({ selected: 'notes' })
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
    await rowIn(held, 'notes').trigger('click')
    expect(held.emitted('select')).toStrictEqual([['notes']])
  })

  it('selects nothing where the row is the selection already', async () => {
    const held = mountTree({ selected: 'notes' })
    await rowIn(held, 'notes').trigger('click')
    expect(held.emitted('select')).toBeUndefined()
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

  it('of the right button asks for a menu where the pointer was', async () => {
    const held = mountTree()
    await rowIn(held, 'notes').trigger('contextmenu', { clientX: 40, clientY: 60 })
    expect(held.emitted('menu')).toStrictEqual([['notes', { x: 40, y: 60 }]])
  })
})

describe('the keyboard', () => {
  const press = (held: Tree, row: string, key: string, over: Record<string, unknown> = {}) =>
    rowIn(held, row).trigger('keydown', { key, ...over })

  it('moves the selection a row at a time', async () => {
    const held = mountTree({ selected: 'work' })
    await press(held, 'work', 'ArrowDown')
    expect(held.emitted('select')).toStrictEqual([['plans']])
  })

  it('opens a shut row with the right arrow, and moves the selection nowhere', async () => {
    const held = mountTree({ selected: 'plans' })
    await press(held, 'plans', 'ArrowRight')

    expect(held.emitted('open')).toStrictEqual([['plans']])
    expect(held.emitted('select')).toBeUndefined()
  })

  it('shuts an open row with the left arrow', async () => {
    const held = mountTree({ selected: 'work' })
    await press(held, 'work', 'ArrowLeft')
    expect(held.emitted('close')).toStrictEqual([['work']])
  })

  it('climbs to the holder from a row that is shut', async () => {
    const held = mountTree({ selected: 'notes' })
    await press(held, 'notes', 'ArrowLeft')

    expect(held.emitted('select')).toStrictEqual([['work']])
    expect(held.emitted('close')).toBeUndefined()
  })

  it('acts on a row that cannot hold, and opens nothing', async () => {
    const held = mountTree({ selected: 'notes' })
    await press(held, 'notes', 'Enter')

    expect(held.emitted('activate')).toStrictEqual([['notes']])
    expect(held.emitted('open')).toBeUndefined()
  })

  it('turns a row that holds as it acts on it', async () => {
    const held = mountTree({ selected: 'plans' })
    await press(held, 'plans', 'Enter')

    expect(held.emitted('activate')).toStrictEqual([['plans']])
    expect(held.emitted('open')).toStrictEqual([['plans']])
  })

  it('reaches the menu the pointer reaches', async () => {
    const held = mountTree({ selected: 'notes' })
    await press(held, 'notes', 'F10', { shiftKey: true })

    expect(held.emitted('menu')).toStrictEqual([['notes', { x: 0, y: 3 * HEIGHT }]])
  })
})

describe('a drag', () => {
  it('moves a row into one that holds', async () => {
    const held = mountTree()
    await dragTo(held, 'loose', 12)
    expect(held.emitted('move')).toStrictEqual([['loose', { into: 'work' }]])
  })

  it('moves a row between two others', async () => {
    const held = mountTree()
    await dragTo(held, 'loose', 2)
    expect(held.emitted('move')).toStrictEqual([['loose', { before: 'work' }]])
  })

  it('is refused into what the row holds, and moves nothing', async () => {
    const held = mountTree()
    await dragTo(held, 'work', HEIGHT + 12)
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
    const held = mountTree({ frame: (run: () => void) => frames.push(run) })

    await dragTo(held, 'loose', 12)
    await rowIn(held, 'loose').trigger('click')

    expect(held.emitted('select')).toBeUndefined()

    frames.forEach((run) => run())
    await rowIn(held, 'loose').trigger('click')
    expect(held.emitted('select')).toStrictEqual([['loose']])
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
    const held = mountTree({ renaming: 'notes', selected: 'notes' })
    await fieldIn(held).trigger('keydown', { key: 'ArrowDown' })

    expect(held.emitted('select')).toBeUndefined()
  })
})

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
import type { Row } from '../lib/row'
import { stubClock } from '@/shared/fixtures/clock'
import type { Clock } from '@/shared/lib/clock'

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

/** A clock whose next frame is now. */
const atOnce: Clock = {
  now: () => 0,
  schedule: (run) => {
    run(0)
    return 0
  },
  cancel: () => {},
}

const mountTree = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(Tree, {
    attachTo: document.body,
    props: { rows: ROWS, open: ['work'], clock: atOnce, ...props },
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
    // The rows and the box around them stand as tall as the rows drawn.
    const list = this.getAttribute('role') === 'tree' || this.classList.contains('tree')
    return boxOf(0, list ? 5 * HEIGHT : 0)
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

const findDragged = (held: Tree) => held.find('.tree__dragged')

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

  it('twice on a row that holds turns it', async () => {
    const held = mountTree()
    await rowIn(held, 'plans').trigger('dblclick')

    expect(held.emitted('open')).toStrictEqual([['plans']])
  })

  it('twice on an open row shuts it', async () => {
    const held = mountTree()
    await rowIn(held, 'work').trigger('dblclick')

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

  it('puts the row the keyboard stands on into the selection', async () => {
    const held = mountTree({ selected: ['work'] })
    await rowIn(held, 'notes').trigger('focus')
    await types(held, 'notes', ' ')

    expect(held.emitted('select')).toStrictEqual([[['work', 'notes']]])
  })

  it('takes that row out again where it already stood in it', async () => {
    const held = mountTree({ selected: ['work', 'notes'] })
    await rowIn(held, 'notes').trigger('focus')
    await types(held, 'notes', ' ')

    expect(held.emitted('select')).toStrictEqual([[['work']]])
  })

  // A space the tree lets by is a space the pane it stands in scrolls under.
  it('keeps the space to itself', () => {
    const held = mountTree({ selected: ['work'] })
    const press = new KeyboardEvent('keydown', { key: ' ', bubbles: true, cancelable: true })
    rowIn(held, 'notes').element.dispatchEvent(press)

    expect(press.defaultPrevented).toBe(true)
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

  it('drags the whole selection, off a row standing in it', async () => {
    const held = mountTree({ selected: ['notes', 'loose'] })
    await dragTo(held, 'loose', 12)

    expect(held.emitted('move')).toStrictEqual([[['notes', 'loose'], { into: 'work' }]])
    expect(held.emitted('select')).toBeUndefined()
  })

  it('selects a row standing outside the selection, and drags it alone', async () => {
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

  it('is refused into what any of the rows dragged holds, and moves nothing', async () => {
    const held = mountTree({ selected: ['work', 'loose'] })
    await dragTo(held, 'loose', HEIGHT + 12)
    expect(held.emitted('move')).toBeUndefined()
  })

  it('takes a row to the top level, let go past the last row', async () => {
    const held = mountTree()
    await dragTo(held, 'work', 5 * HEIGHT - 2)
    expect(held.emitted('move')).toStrictEqual([[['work'], { into: null }]])
  })

  // The pointer is caught on the window, so a row let go over another pane is
  // let go somewhere the tree does not answer for.
  it('moves nothing let go beside the tree', async () => {
    const held = mountTree()
    const from = rowIn(held, 'work').element

    pointer('pointerdown', middleOf(held, 'work'), from)
    pointer('pointermove', 2 * HEIGHT, window, { clientX: 400 })
    pointer('pointerup', 2 * HEIGHT, window, { clientX: 400 })
    await held.vm.$nextTick()

    expect(held.emitted('move')).toBeUndefined()
  })

  it('moves nothing let go below the tree', async () => {
    const held = mountTree()
    await dragTo(held, 'work', 9 * HEIGHT)

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
    const world = stubClock()
    const held = mountTree({ selected: ['loose'], clock: world.clock })

    await dragTo(held, 'loose', 12)
    await rowIn(held, 'notes').trigger('click')

    expect(held.emitted('select')).toBeUndefined()

    world.run()
    await rowIn(held, 'notes').trigger('click')
    expect(held.emitted('select')).toStrictEqual([[['notes']]])
  })
})

describe('rows dragged out of the tree', () => {
  it('says which rows are being dragged, once for the whole drag', async () => {
    const held = mountTree({ selected: ['notes', 'loose'] })
    const from = rowIn(held, 'loose').element

    pointer('pointerdown', middleOf(held, 'loose'), from)
    pointer('pointermove', 3 * HEIGHT)
    pointer('pointermove', 4 * HEIGHT)
    await held.vm.$nextTick()

    expect(held.emitted('drag')).toStrictEqual([[['notes', 'loose']]])
  })

  it('says so for a drag over another pane, where nothing in the tree is landed on', async () => {
    const held = mountTree()
    const from = rowIn(held, 'work').element

    pointer('pointerdown', middleOf(held, 'work'), from)
    pointer('pointermove', 2 * HEIGHT, window, { clientX: 400 })
    pointer('pointerup', 2 * HEIGHT, window, { clientX: 400 })
    await held.vm.$nextTick()

    expect(held.emitted('drag')).toStrictEqual([[['work']]])
    expect(held.emitted('drop')).toStrictEqual([[]])
    expect(held.emitted('move')).toBeUndefined()
  })

  it('says it has let go after saying where the rows landed', async () => {
    const held = mountTree()
    await dragTo(held, 'loose', 12)

    expect(held.emitted('move')).toStrictEqual([[['loose'], { into: 'work' }]])
    expect(held.emitted('drop')).toStrictEqual([[]])
  })

  it('drags nothing where the pointer did not travel far enough', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 110)
    pointer('pointerup', 110)
    await held.vm.$nextTick()

    expect(held.emitted('drag')).toBeUndefined()
    expect(held.emitted('drop')).toBeUndefined()
  })
})

describe('what follows the pointer', () => {
  it('says the name of the one row being dragged', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(findDragged(held).text()).toBe('Loose')
  })

  it('says how many are being dragged, where there are several', async () => {
    const held = mountTree({ selected: ['notes', 'loose'] })
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(findDragged(held).text()).toBe('2 rows')
  })

  it('says it in the words the caller gave for how many', async () => {
    const held = mountTree({
      selected: ['notes', 'loose'],
      counted: (rows: number) => `${rows} files`,
    })
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(findDragged(held).text()).toBe('2 files')
  })

  it('stands where the pointer is', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 60)
    await held.vm.$nextTick()

    expect(findDragged(held).attributes('style')).toContain('top: 60px')
  })

  it('is drawn nowhere before the pointer has travelled far enough', async () => {
    const held = mountTree()
    pointer('pointerdown', 108, rowIn(held, 'loose').element)
    pointer('pointermove', 110)
    await held.vm.$nextTick()

    expect(findDragged(held).exists()).toBe(false)
  })

  it('is drawn nowhere once the rows have been let go of', async () => {
    const held = mountTree()
    await dragTo(held, 'loose', 12)

    expect(findDragged(held).exists()).toBe(false)
  })

  it('is drawn nowhere at all while nothing is being dragged', () => {
    expect(findDragged(mountTree()).exists()).toBe(false)
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

describe('the attribute rows are marked with', () => {
  const ATTRIBUTE = 'data-somewhere'

  const markIn = (held: Tree, row: string) => rowIn(held, row).attributes(ATTRIBUTE)

  /** The place a row stands for, or the place it sits in. */
  const valueFor = (row: string | null): string => {
    if (row === null) return ''
    if (row === 'work' || row === 'plans' || row === 'empty') return row
    return row === 'friday' ? 'plans' : 'work'
  }

  it('is written on each row, and on the tree for the top level', () => {
    const held = mountTree({ marking: { attribute: ATTRIBUTE, valueFor } })

    expect(markIn(held, 'plans')).toBe('plans')
    expect(markIn(held, 'notes')).toBe('work')
    expect(held.get('.tree').attributes(ATTRIBUTE)).toBe('')
  })

  it('is written nowhere while the tree is given no marking', () => {
    const held = mountTree()

    expect(markIn(held, 'plans')).toBeUndefined()
    expect(held.get('.tree').attributes(ATTRIBUTE)).toBeUndefined()
  })

  it('is written under the name it was given', () => {
    const held = mountTree({ marking: { attribute: 'data-elsewhere', valueFor } })

    expect(rowIn(held, 'plans').attributes('data-elsewhere')).toBe('plans')
    expect(markIn(held, 'plans')).toBeUndefined()
  })

  it('is written on the rows answered for and on no other', () => {
    const marking = {
      attribute: ATTRIBUTE,
      valueFor: (row: string | null) => (row === 'empty' ? 'empty' : null),
    }
    const held = mountTree({ marking })

    expect(markIn(held, 'empty')).toBe('empty')
    expect(markIn(held, 'plans')).toBeUndefined()
  })
})

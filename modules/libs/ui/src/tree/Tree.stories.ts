/**
 * Every situation the tree has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * What a row stands for is a plain glyph here. The tree knows nothing about
 * what it draws, and neither do these.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { ref, watch } from 'vue'
import Tree from './Tree.vue'
import type { RowLanding, Row, RowId } from './row'

/** A row taken out of wherever it stands. */
const without = (rows: readonly Row[], id: RowId): readonly Row[] =>
  rows
    .filter((row) => row.id !== id)
    .map((row) => (row.rows ? { ...row, rows: without(row.rows, id) } : row))

const found = (rows: readonly Row[], id: RowId): Row | null => {
  for (const row of rows) {
    if (row.id === id) return row
    const below = found(row.rows ?? [], id)
    if (below) return below
  }
  return null
}

/** Rows put where a landing says, in the order they were dragged. */
const put = (rows: readonly Row[], at: RowLanding, held: readonly Row[]): readonly Row[] =>
  rows.flatMap((row) => {
    const below = row.rows ? { ...row, rows: put(row.rows, at, held) } : row
    if ('before' in at && row.id === at.before) return [...held, below]
    if ('into' in at && row.id === at.into) {
      return [{ ...below, rows: [...(below.rows ?? []), ...held] }]
    }
    return [below]
  })

/** The application's part: what a move comes to, in the rows it holds. */
const moved = (rows: readonly Row[], dragged: readonly RowId[], at: RowLanding): readonly Row[] => {
  const held = dragged.map((row) => found(rows, row)).filter((row): row is Row => row !== null)
  const left = dragged.reduce((rest, row) => without(rest, row), rows)
  return held.length ? put(left, at, held) : rows
}

/** The application's part again: rows taken out of the tree. */
const removed = (rows: readonly Row[], dragged: readonly RowId[]): readonly Row[] =>
  dragged.reduce((rest, row) => without(rest, row), rows)

/** The application's part again: a row under a new name. */
const renamed = (rows: readonly Row[], row: RowId, name: string): readonly Row[] =>
  rows.map((each) => ({
    ...each,
    ...(each.id === row ? { name } : {}),
    ...(each.rows ? { rows: renamed(each.rows, row, name) } : {}),
  }))

const FEW: readonly Row[] = [
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

/** One row inside the next, as far down as it goes. */
const deep = (levels: number): readonly Row[] => {
  const at = (level: number): readonly Row[] =>
    level > levels
      ? [{ id: 'bottom', name: 'The bottom', holds: false }]
      : [{ id: `level-${level}`, name: `Level ${level}`, holds: true, rows: at(level + 1) }]
  return at(1)
}

const DEEP_LEVELS = 14

const many = (count: number): readonly Row[] =>
  Array.from({ length: count }, (_, at) => ({
    id: `row-${at}`,
    name: `Row ${at + 1}`,
    holds: false,
  }))

const CYRILLIC: readonly Row[] = [
  {
    id: 'грядки',
    name: 'Грядки',
    holds: true,
    rows: [
      { id: 'морковь', name: 'Морковь', holds: false },
      { id: 'семена', name: 'Список семян на весну', holds: false },
    ],
  },
  { id: 'сарай', name: 'Сарай', holds: true },
]

const UNBROKEN: readonly Row[] = [
  {
    id: 'run-on',
    name: 'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakat',
    holds: false,
  },
  {
    id: 'long',
    name: 'A name that goes on and on, well past the width anything drawing it is likely to have',
    holds: false,
  },
]

const NAMELESS: readonly Row[] = [
  { id: 'blank', name: '', holds: false },
  { id: 'blank-holder', name: '', holds: true, rows: [{ id: 'inside', name: '', holds: false }] },
  { id: 'named', name: 'Named', holds: false },
]

interface Corpus {
  readonly rows: readonly Row[]
  readonly open: readonly RowId[]
}

/** What a reader can start from. */
const CORPORA = {
  'a few': { rows: FEW, open: ['work'] },
  nested: {
    rows: deep(DEEP_LEVELS),
    open: Array.from({ length: DEEP_LEVELS }, (_, at) => `level-${at + 1}`),
  },
  many: { rows: many(2000), open: [] },
  'other scripts': { rows: CYRILLIC, open: ['грядки'] },
  unbroken: { rows: UNBROKEN, open: [] },
  nameless: { rows: NAMELESS, open: ['blank-holder'] },
  one: { rows: [{ id: 'only', name: 'Only', holds: false }], open: [] },
  empty: { rows: [], open: [] },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  threshold: number
  name: string
  /** The row whose name starts out in a field. */
  renaming: RowId | null
  /** The rows the selection starts out on. */
  selected: readonly RowId[]
  /** Given by the story, and nothing a reader turns. */
  rows?: never
  open?: never
  counted?: never
  clock?: never
}

const meta: Meta<Knobs> = {
  title: 'Tree',
  component: Tree,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    corpus: {
      control: 'select',
      options: Object.keys(CORPORA),
      description: 'What the tree starts as. Changing it starts afresh.',
    },
    threshold: { control: { type: 'range', min: 0, max: 24, step: 1 } },
    name: { control: 'text' },
    renaming: { control: 'text' },
    selected: { control: 'object' },
    rows: { table: { disable: true } },
    open: { table: { disable: true } },
    counted: { table: { disable: true } },
    clock: { table: { disable: true } },
  },
  args: { corpus: 'a few', threshold: 4, name: 'Tree', renaming: null, selected: [] },
  render: (args) => ({
    components: { Tree },
    setup() {
      const rows = ref<readonly Row[]>(CORPORA[args.corpus].rows)
      const open = ref<readonly RowId[]>(CORPORA[args.corpus].open)
      const selected = ref<readonly RowId[]>(args.selected)
      const renaming = ref<RowId | null>(args.renaming)

      watch(
        () => args.corpus,
        (next) => {
          rows.value = CORPORA[next].rows
          open.value = CORPORA[next].open
          selected.value = []
        },
      )

      return {
        args,
        rows,
        open,
        selected,
        renaming,
        onOpen: (row: RowId) => {
          open.value = [...open.value, row]
        },
        onClose: (row: RowId) => {
          open.value = open.value.filter((each) => each !== row)
        },
        onSelect: (picked: readonly RowId[]) => {
          selected.value = picked
        },
        onMove: (dragged: readonly RowId[], at: RowLanding) => {
          rows.value = moved(rows.value, dragged, at)
        },
        onRemove: (dragged: readonly RowId[]) => {
          rows.value = removed(rows.value, dragged)
          selected.value = []
        },
        onRename: (row: RowId, name: string) => {
          rows.value = renamed(rows.value, row, name)
        },
      }
    },
    template: `
      <div style="height: 100vh; width: 18rem">
        <Tree
          v-model:renaming="renaming"
          :rows="rows"
          :open="open"
          :selected="selected"
          :threshold="args.threshold"
          :name="args.name"
          @open="onOpen"
          @close="onClose"
          @select="onSelect"
          @move="onMove"
          @remove="onRemove"
          @rename="onRename"
        >
          <template #icon="{ holds, open: shown }">
            <span aria-hidden="true">{{ holds ? (shown ? '▤' : '▥') : '▭' }}</span>
          </template>
        </Tree>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A folder open, a folder shut, a folder holding nothing, and a loose row. */
export const AFew: Story = {}

/** Fourteen levels, each inside the one above. */
export const Nested: Story = { args: { corpus: 'nested' } }

/** Two thousand rows, far more than the window has room for. */
export const Many: Story = { args: { corpus: 'many' } }

/** Names that are not Latin. */
export const OtherScripts: Story = { args: { corpus: 'other scripts' } }

/** A name far too long, and one with nothing in it to break at. */
export const Unbroken: Story = { args: { corpus: 'unbroken' } }

/** Rows whose names are the empty string. */
export const Nameless: Story = { args: { corpus: 'nameless' } }

/** One row, and nothing to move it among. */
export const One: Story = { args: { corpus: 'one' } }

/** Nothing at all. */
export const Empty: Story = { args: { corpus: 'empty' } }

/** A name in a field, over the row it belongs to. */
export const Renaming: Story = { args: { renaming: 'notes' } }

/** Several rows selected at once. */
export const Several: Story = { args: { selected: ['work', 'notes', 'loose'] } }

/** A selection running from inside a folder out past the end of it. */
export const AcrossAFolder: Story = { args: { selected: ['plans', 'notes', 'empty'] } }

const rowIn = (canvas: HTMLElement, row: string) => {
  const held = canvas.querySelector<HTMLElement>(`[data-tree-row="${row}"]`)
  if (!held) throw new Error(`no row called ${row}`)
  return held
}

const drawn = (canvas: HTMLElement) =>
  [...canvas.querySelectorAll('[data-tree-row]')].map((row) => row.getAttribute('data-tree-row'))

/** The rows the tree announces as selected, in the order they are drawn. */
const selectedIn = (canvas: HTMLElement) =>
  [...canvas.querySelectorAll('[aria-selected="true"]')].map((row) =>
    row.getAttribute('data-tree-row'),
  )

const middleOf = (element: Element) => {
  const box = element.getBoundingClientRect()
  return { clientX: box.x + box.width / 2, clientY: box.y + box.height / 2 }
}

/** A row picked up and let go somewhere, in as many steps as a hand takes. */
async function dragTo(from: Element, to: { clientX: number; clientY: number }): Promise<void> {
  const at = middleOf(from)
  await userEvent.pointer([
    { keys: '[MouseLeft>]', target: from, coords: at },
    { coords: { clientX: (at.clientX + to.clientX) / 2, clientY: (at.clientY + to.clientY) / 2 } },
    { coords: to },
    { keys: '[/MouseLeft]' },
  ])
}

/** The arrows open a row, walk into it, and shut it again. */
export const WalksWithTheKeyboard: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    rowIn(canvasElement, 'plans').focus()

    await userEvent.keyboard('{ArrowRight}')
    await expect(drawn(canvasElement)).toContain('friday')
    await expect(rowIn(canvasElement, 'plans').getAttribute('aria-expanded')).toBe('true')

    await userEvent.keyboard('{ArrowRight}')
    await expect(document.activeElement).toBe(rowIn(canvasElement, 'friday'))
    await expect(rowIn(canvasElement, 'friday').getAttribute('aria-selected')).toBe('true')

    await userEvent.keyboard('{ArrowLeft}{ArrowLeft}')
    await expect(drawn(canvasElement)).not.toContain('friday')
    await expect(document.activeElement).toBe(rowIn(canvasElement, 'plans'))
  },
}

/** Dragging a row over the middle of one that holds puts it inside. */
export const DragsIntoARow: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    await dragTo(rowIn(canvasElement, 'loose'), middleOf(rowIn(canvasElement, 'work')))

    await expect(rowIn(canvasElement, 'loose').getAttribute('aria-level')).toBe('2')
    await expect(drawn(canvasElement)).toStrictEqual(['work', 'plans', 'notes', 'loose', 'empty'])
  },
}

/** Dragging a row over the end of another puts it between. */
export const DragsBetweenRows: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const first = rowIn(canvasElement, 'work').getBoundingClientRect()

    await dragTo(rowIn(canvasElement, 'loose'), {
      clientX: first.x + first.width / 2,
      clientY: first.y + 1,
    })

    await expect(drawn(canvasElement)).toStrictEqual(['loose', 'work', 'plans', 'notes', 'empty'])
    await expect(rowIn(canvasElement, 'loose').getAttribute('aria-level')).toBe('1')
  },
}

/** A press with the join key held takes a row into the selection, and out of it again. */
export const JoinsARow: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    // One hand, so what it holds down is still held down at the press.
    const hand = userEvent.setup()
    await hand.click(rowIn(canvasElement, 'work'))

    await hand.keyboard('{Control>}')
    await hand.click(rowIn(canvasElement, 'loose'))
    await expect(selectedIn(canvasElement)).toStrictEqual(['work', 'loose'])

    await hand.click(rowIn(canvasElement, 'loose'))
    await expect(selectedIn(canvasElement)).toStrictEqual(['work'])
    await hand.keyboard('{/Control}')
  },
}

/** A press with Shift held reaches from the anchor, wherever the rows are drawn. */
export const ReachesToARow: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const hand = userEvent.setup()
    await hand.click(rowIn(canvasElement, 'plans'))

    await hand.keyboard('{Shift>}')
    await hand.click(rowIn(canvasElement, 'empty'))
    await expect(selectedIn(canvasElement)).toStrictEqual(['plans', 'notes', 'empty'])

    await hand.click(rowIn(canvasElement, 'work'))
    await expect(selectedIn(canvasElement)).toStrictEqual(['work', 'plans'])
    await hand.keyboard('{/Shift}')
  },
}

/** Two rows dragged at once, held part way to the folder they are going into. */
export const CarryingSeveral: Story = {
  args: { selected: ['notes', 'loose'] },
  play: async ({ canvasElement }) => {
    const from = rowIn(canvasElement, 'loose')
    const at = middleOf(from)
    const to = middleOf(rowIn(canvasElement, 'work'))

    await userEvent.pointer([
      { keys: '[MouseLeft>]', target: from, coords: at },
      { coords: { clientX: at.clientX, clientY: (at.clientY + to.clientY) / 2 } },
      { coords: to },
    ])

    await expect(canvasElement.querySelector('.tree__dragged')?.textContent?.trim()).toBe('2 rows')
    await expect(rowIn(canvasElement, 'work').getAttribute('data-into')).toBe('true')
  },
}

/** Two rows let go over a folder land in it together. */
export const DragsSeveralIntoARow: Story = {
  tags: ['!dev'],
  args: { selected: ['notes', 'loose'] },
  play: async ({ canvasElement }) => {
    await dragTo(rowIn(canvasElement, 'loose'), middleOf(rowIn(canvasElement, 'work')))

    await expect(drawn(canvasElement)).toStrictEqual(['work', 'plans', 'notes', 'loose', 'empty'])
    await expect(rowIn(canvasElement, 'loose').getAttribute('aria-level')).toBe('2')
    await expect(canvasElement.querySelector('.tree__dragged')).toBeNull()
  },
}

/** A row cannot be dragged inside what it holds, and letting go moves nothing. */
export const RefusesWhatItHolds: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const before = drawn(canvasElement)

    await dragTo(rowIn(canvasElement, 'work'), middleOf(rowIn(canvasElement, 'plans')))

    await expect(drawn(canvasElement)).toStrictEqual(before)
    await expect(rowIn(canvasElement, 'work').getAttribute('aria-level')).toBe('1')
  },
}

/** A name typed over a row, and taken. */
export const TakesAName: Story = {
  tags: ['!dev'],
  args: { renaming: 'notes' },
  play: async ({ canvasElement }) => {
    const field = canvasElement.querySelector<HTMLInputElement>('.tree__field')
    if (!field) throw new Error('no field')

    await userEvent.clear(field)
    await userEvent.type(field, 'Грядки{Enter}')

    await expect(canvasElement.querySelector('.tree__field')).toBeNull()
    await expect(rowIn(canvasElement, 'notes').textContent).toContain('Грядки')
  },
}

/** A name typed over a row, and abandoned. */
export const AbandonsAName: Story = {
  tags: ['!dev'],
  args: { renaming: 'notes' },
  play: async ({ canvasElement }) => {
    const field = canvasElement.querySelector<HTMLInputElement>('.tree__field')
    if (!field) throw new Error('no field')

    await userEvent.clear(field)
    await userEvent.type(field, 'Something else{Escape}')

    await expect(canvasElement.querySelector('.tree__field')).toBeNull()
    await expect(rowIn(canvasElement, 'notes').textContent).toContain('Notes')
  },
}

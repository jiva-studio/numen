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
import type { Row, RowId } from '../lib/row'
import type { RowDropTarget } from '../lib/drop'

/** A row taken out of wherever it stands. */
const removeRow = (rows: readonly Row[], id: RowId): readonly Row[] =>
  rows
    .filter((row) => row.id !== id)
    .map((row) => (row.rows ? { ...row, rows: removeRow(row.rows, id) } : row))

const found = (rows: readonly Row[], id: RowId): Row | null => {
  for (const row of rows) {
    if (row.id === id) return row
    const below = found(row.rows ?? [], id)
    if (below) return below
  }
  return null
}

/** Rows put where a drop target says, in the order they were dragged. */
const insertRows = (
  rows: readonly Row[],
  target: RowDropTarget,
  dragRows: readonly Row[],
): readonly Row[] =>
  rows.flatMap((row) => {
    const below = row.rows ? { ...row, rows: insertRows(row.rows, target, dragRows) } : row
    if ('before' in target && row.id === target.before) return [...dragRows, below]
    if ('into' in target && row.id === target.into) {
      return [{ ...below, rows: [...(below.rows ?? []), ...dragRows] }]
    }
    return [below]
  })

/** The application's part: what a move evaluates to, in the rows it holds. */
const moveRows = (
  rows: readonly Row[],
  dragIds: readonly RowId[],
  dropTarget: RowDropTarget,
): readonly Row[] => {
  const target = dragIds.map((row) => found(rows, row)).filter((row): row is Row => row !== null)
  const remainingRows = dragIds.reduce((rest, row) => removeRow(rest, row), rows)
  return target.length ? insertRows(remainingRows, dropTarget, target) : rows
}

/** The application's part again: rows taken out of the tree. */
const removeRows = (rows: readonly Row[], dragIds: readonly RowId[]): readonly Row[] =>
  dragIds.reduce((rest, row) => removeRow(rest, row), rows)

/** The application's part again: a row under a new name. */
const renameRow = (rows: readonly Row[], row: RowId, name: string): readonly Row[] =>
  rows.map((each) => ({
    ...each,
    ...(each.id === row ? { name } : {}),
    ...(each.rows ? { rows: renameRow(each.rows, row, name) } : {}),
  }))

const FEW: readonly Row[] = [
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

/** One row inside the next, as far down as it goes. */
const deep = (levels: number): readonly Row[] => {
  const createLevel = (level: number): readonly Row[] =>
    level > levels
      ? [{ id: 'bottom', name: 'The bottom', hasChildren: false }]
      : [
          {
            id: `level-${level}`,
            name: `Level ${level}`,
            hasChildren: true,
            rows: createLevel(level + 1),
          },
        ]
  return createLevel(1)
}

const DEEP_LEVELS = 14

const many = (count: number): readonly Row[] =>
  Array.from({ length: count }, (_, index) => ({
    id: `row-${index}`,
    name: `Row ${index + 1}`,
    hasChildren: false,
  }))

const CYRILLIC: readonly Row[] = [
  {
    id: 'грядки',
    name: 'Грядки',
    hasChildren: true,
    rows: [
      { id: 'морковь', name: 'Морковь', hasChildren: false },
      { id: 'семена', name: 'Список семян на весну', hasChildren: false },
    ],
  },
  { id: 'сарай', name: 'Сарай', hasChildren: true },
]

const UNBROKEN: readonly Row[] = [
  {
    id: 'run-on',
    name: 'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakat',
    hasChildren: false,
  },
  {
    id: 'long',
    name: 'A name that goes on and on, well past the width anything drawing it is likely to have',
    hasChildren: false,
  },
]

const NAMELESS: readonly Row[] = [
  { id: 'blank', name: '', hasChildren: false },
  {
    id: 'blank-holder',
    name: '',
    hasChildren: true,
    rows: [{ id: 'inside', name: '', hasChildren: false }],
  },
  { id: 'named', name: 'Named', hasChildren: false },
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
    open: Array.from({ length: DEEP_LEVELS }, (_, index) => `level-${index + 1}`),
  },
  many: { rows: many(2000), open: [] },
  'other scripts': { rows: CYRILLIC, open: ['грядки'] },
  unbroken: { rows: UNBROKEN, open: [] },
  nameless: { rows: NAMELESS, open: ['blank-holder'] },
  one: { rows: [{ id: 'only', name: 'Only', hasChildren: false }], open: [] },
  empty: { rows: [], open: [] },
} satisfies Record<string, Corpus>

type Corpora = keyof typeof CORPORA

interface Knobs {
  corpus: Corpora
  threshold: number
  name: string
  /** The row whose name starts out in a field. */
  renamingPath: RowId | null
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
    renamingPath: { control: 'text' },
    selected: { control: 'object' },
    rows: { table: { disable: true } },
    open: { table: { disable: true } },
    counted: { table: { disable: true } },
    clock: { table: { disable: true } },
  },
  args: { corpus: 'a few', threshold: 4, name: 'Tree', renamingPath: null, selected: [] },
  render: (args) => ({
    components: { Tree },
    setup() {
      const rows = ref<readonly Row[]>(CORPORA[args.corpus].rows)
      const open = ref<readonly RowId[]>(CORPORA[args.corpus].open)
      const selected = ref<readonly RowId[]>(args.selected)
      const renamingPath = ref<RowId | null>(args.renamingPath)

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
        renamingPath,
        onOpen: (row: RowId) => {
          open.value = [...open.value, row]
        },
        onClose: (row: RowId) => {
          open.value = open.value.filter((each) => each !== row)
        },
        onSelect: (picked: readonly RowId[]) => {
          selected.value = picked
        },
        onMove: (dragged: readonly RowId[], target: RowDropTarget) => {
          rows.value = moveRows(rows.value, dragged, target)
        },
        onRemove: (dragged: readonly RowId[]) => {
          rows.value = removeRows(rows.value, dragged)
          selected.value = []
        },
        onRename: (row: RowId, name: string) => {
          rows.value = renameRow(rows.value, row, name)
        },
      }
    },
    template: `
      <div style="height: 100vh; width: 18rem">
        <Tree
          v-model:renamingPath="renamingPath"
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
export const Renaming: Story = { args: { renamingPath: 'notes' } }

/** Several rows selected at once. */
export const Several: Story = { args: { selected: ['work', 'notes', 'loose'] } }

/** A selection running from inside a folder out past the end of it. */
export const AcrossAFolder: Story = { args: { selected: ['plans', 'notes', 'empty'] } }

const getRow = (canvas: HTMLElement, row: string) => {
  const element = canvas.querySelector<HTMLElement>(`[data-tree-row="${row}"]`)
  if (!element) throw new Error(`no row called ${row}`)
  return element
}

const getDrawnRows = (canvas: HTMLElement) =>
  [...canvas.querySelectorAll('[data-tree-row]')].map((row) => row.getAttribute('data-tree-row'))

/** The rows the tree announces as selected, in the order they are drawn. */
const getSelected = (canvas: HTMLElement) =>
  [...canvas.querySelectorAll('[aria-selected="true"]')].map((row) =>
    row.getAttribute('data-tree-row'),
  )

const getMiddle = (element: Element) => {
  const box = element.getBoundingClientRect()
  return { clientX: box.x + box.width / 2, clientY: box.y + box.height / 2 }
}

/** A row picked up and let go somewhere, in as many steps as a hand takes. */
async function dragTo(from: Element, to: { clientX: number; clientY: number }): Promise<void> {
  const start = getMiddle(from)
  await userEvent.pointer([
    { keys: '[MouseLeft>]', target: from, coords: start },
    {
      coords: {
        clientX: (start.clientX + to.clientX) / 2,
        clientY: (start.clientY + to.clientY) / 2,
      },
    },
    { coords: to },
    { keys: '[/MouseLeft]' },
  ])
}

/** The arrows open a row, walk into it, and shut it again. */
export const WalksWithTheKeyboard: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    getRow(canvasElement, 'plans').focus()

    await userEvent.keyboard('{ArrowRight}')
    await expect(getDrawnRows(canvasElement)).toContain('friday')
    await expect(getRow(canvasElement, 'plans').getAttribute('aria-expanded')).toBe('true')

    await userEvent.keyboard('{ArrowRight}')
    await expect(document.activeElement).toBe(getRow(canvasElement, 'friday'))
    await expect(getRow(canvasElement, 'friday').getAttribute('aria-selected')).toBe('true')

    await userEvent.keyboard('{ArrowLeft}{ArrowLeft}')
    await expect(getDrawnRows(canvasElement)).not.toContain('friday')
    await expect(document.activeElement).toBe(getRow(canvasElement, 'plans'))
  },
}

/** Dragging a row over the middle of one that holds puts it inside. */
export const DragsIntoARow: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    await dragTo(getRow(canvasElement, 'loose'), getMiddle(getRow(canvasElement, 'work')))

    await expect(getRow(canvasElement, 'loose').getAttribute('aria-level')).toBe('2')
    await expect(getDrawnRows(canvasElement)).toStrictEqual([
      'work',
      'plans',
      'notes',
      'loose',
      'empty',
    ])
  },
}

/** Dragging a row over the end of another puts it between. */
export const DragsBetweenRows: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const first = getRow(canvasElement, 'work').getBoundingClientRect()

    await dragTo(getRow(canvasElement, 'loose'), {
      clientX: first.x + first.width / 2,
      clientY: first.y + 1,
    })

    await expect(getDrawnRows(canvasElement)).toStrictEqual([
      'loose',
      'work',
      'plans',
      'notes',
      'empty',
    ])
    await expect(getRow(canvasElement, 'loose').getAttribute('aria-level')).toBe('1')
  },
}

/** A press with the join key held takes a row into the selection, and out of it again. */
export const JoinsARow: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    // One hand, so what it holds down is still held down at the press.
    const hand = userEvent.setup()
    await hand.click(getRow(canvasElement, 'work'))

    await hand.keyboard('{Control>}')
    await hand.click(getRow(canvasElement, 'loose'))
    await expect(getSelected(canvasElement)).toStrictEqual(['work', 'loose'])

    await hand.click(getRow(canvasElement, 'loose'))
    await expect(getSelected(canvasElement)).toStrictEqual(['work'])
    await hand.keyboard('{/Control}')
  },
}

/** A press with Shift held reaches from the anchor, wherever the rows are drawn. */
export const ReachesToARow: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const hand = userEvent.setup()
    await hand.click(getRow(canvasElement, 'plans'))

    await hand.keyboard('{Shift>}')
    await hand.click(getRow(canvasElement, 'empty'))
    await expect(getSelected(canvasElement)).toStrictEqual(['plans', 'notes', 'empty'])

    await hand.click(getRow(canvasElement, 'work'))
    await expect(getSelected(canvasElement)).toStrictEqual(['work', 'plans'])
    await hand.keyboard('{/Shift}')
  },
}

/** Two rows dragged at once, held part way to the folder they are going into. */
export const DraggingSeveral: Story = {
  args: { selected: ['notes', 'loose'] },
  play: async ({ canvasElement }) => {
    const from = getRow(canvasElement, 'loose')
    const start = getMiddle(from)
    const to = getMiddle(getRow(canvasElement, 'work'))

    await userEvent.pointer([
      { keys: '[MouseLeft>]', target: from, coords: start },
      { coords: { clientX: start.clientX, clientY: (start.clientY + to.clientY) / 2 } },
      { coords: to },
    ])

    await expect(canvasElement.querySelector('.tree__dragged')?.textContent?.trim()).toBe('2 rows')
    await expect(getRow(canvasElement, 'work').getAttribute('data-into')).toBe('true')
  },
}

/** Two rows let go over a folder land in it together. */
export const DragsSeveralIntoARow: Story = {
  tags: ['!dev'],
  args: { selected: ['notes', 'loose'] },
  play: async ({ canvasElement }) => {
    await dragTo(getRow(canvasElement, 'loose'), getMiddle(getRow(canvasElement, 'work')))

    await expect(getDrawnRows(canvasElement)).toStrictEqual([
      'work',
      'plans',
      'notes',
      'loose',
      'empty',
    ])
    await expect(getRow(canvasElement, 'loose').getAttribute('aria-level')).toBe('2')
    await expect(canvasElement.querySelector('.tree__dragged')).toBeNull()
  },
}

/** A row cannot be dragged inside what it holds, and letting go moves nothing. */
export const RefusesWhatItHolds: Story = {
  tags: ['!dev'],
  play: async ({ canvasElement }) => {
    const before = getDrawnRows(canvasElement)

    await dragTo(getRow(canvasElement, 'work'), getMiddle(getRow(canvasElement, 'plans')))

    await expect(getDrawnRows(canvasElement)).toStrictEqual(before)
    await expect(getRow(canvasElement, 'work').getAttribute('aria-level')).toBe('1')
  },
}

/** A name typed over a row, and taken. */
export const TakesAName: Story = {
  tags: ['!dev'],
  args: { renamingPath: 'notes' },
  play: async ({ canvasElement }) => {
    const field = canvasElement.querySelector<HTMLInputElement>('.tree__field')
    if (!field) throw new Error('no field')

    await userEvent.clear(field)
    await userEvent.type(field, 'Грядки{Enter}')

    await expect(canvasElement.querySelector('.tree__field')).toBeNull()
    await expect(getRow(canvasElement, 'notes').textContent).toContain('Грядки')
  },
}

/** A name typed over a row, and abandoned. */
export const AbandonsAName: Story = {
  tags: ['!dev'],
  args: { renamingPath: 'notes' },
  play: async ({ canvasElement }) => {
    const field = canvasElement.querySelector<HTMLInputElement>('.tree__field')
    if (!field) throw new Error('no field')

    await userEvent.clear(field)
    await userEvent.type(field, 'Something else{Escape}')

    await expect(canvasElement.querySelector('.tree__field')).toBeNull()
    await expect(getRow(canvasElement, 'notes').textContent).toContain('Notes')
  },
}

/**
 * Every state a searchable list has to survive, in the light token set and the
 * dark one. Also the test corpus: each story is run in a browser by
 * `@storybook/addon-vitest`.
 *
 * The frame is one container: the search line first, a rule, and the rows in
 * the same frame, so what is judged here is what every list that is searched
 * looks like.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import SearchableList from './SearchableList.vue'
import SearchableListPopup from './SearchableListPopup.vue'
import type { ListItem } from './rows'

interface Knobs {
  items: readonly ListItem[]
  label: string
  placeholder: string
  emptyText: string
  maxRows: number
  isPending: boolean
  isFiltered: boolean
  hasAction: boolean
  needsQuery: boolean
  modelValue: string | null
  onSelect: (id: string) => void
  onAct: (query: string) => void
  onClose: () => void
}

const MODELS: readonly ListItem[] = [
  { id: 'quince-4', text: 'Quince 4 Argon', group: 'Meadow' },
  { id: 'quince-3', text: 'Quince 3.8 Flash', group: 'Meadow' },
  { id: 'plum', text: 'Orchard Plum 5.5', group: 'Meadow' },
  { id: 'default', text: 'Default (recommended)', group: 'Orchard' },
  { id: 'pear', text: 'Pear 5.5', group: 'Orchard', detail: 'Fast, for everyday work' },
  { id: 'fig', text: 'Fig 5.5', group: 'Orchard' },
]

const SHELVES: readonly ListItem[] = Array.from({ length: 60 }, (_, index) => ({
  id: `shelf-${index}`,
  text: `Shelf ${index + 1}`,
}))

const meta: Meta<Knobs> = {
  title: 'Controls/SearchableList',
  parameters: { layout: 'centered' },
  args: {
    items: MODELS,
    label: 'Models',
    placeholder: 'Search models',
    emptyText: 'No models found',
    maxRows: 8,
    isPending: false,
    isFiltered: true,
    hasAction: false,
    needsQuery: false,
    modelValue: null,
    onSelect: fn(),
    onAct: fn(),
    onClose: fn(),
  },
  render: (args) => ({
    components: { SearchableList },
    setup() {
      const current = ref(args.modelValue)
      return { args, current }
    },
    template: `
      <div class="numen" style="padding:2rem;inline-size:22rem;background:var(--numen-surface);color:var(--numen-ink)">
        <SearchableList
          v-model="current"
          :items="args.items"
          :label="args.label"
          :placeholder="args.placeholder"
          :empty-text="args.emptyText"
          :max-rows="args.maxRows"
          :is-pending="args.isPending"
          :is-filtered="args.isFiltered"
          :has-action="args.hasAction"
          :needs-query="args.needsQuery"
          @select="args.onSelect"
          @act="args.onAct"
          @close="args.onClose"
        >
          <template #action="{ query }">Create "{{ query }}"</template>
        </SearchableList>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getFrame = (canvasElement: HTMLElement) =>
  canvasElement.querySelector<HTMLElement>('[data-numen-searchable-list]')!

const expectOneFrame = async (canvasElement: HTMLElement) => {
  const frame = getFrame(canvasElement)
  const canvas = within(canvasElement)
  await expect(frame.firstElementChild).toBe(canvas.getByRole('combobox'))
  await expect(frame.contains(canvas.getByRole('listbox'))).toBe(true)
}

/** Rows under the names of their groups, one of them in force. */
export const Grouped: Story = {
  args: { modelValue: 'pear' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expectOneFrame(canvasElement)
    await expect(canvas.getAllByRole('option')).toHaveLength(6)
    await expect(canvasElement.querySelectorAll('.searchable-list__heading')).toHaveLength(2)
    await expect(canvas.getByRole('option', { name: /Pear 5.5/ })).toHaveAttribute(
      'aria-current',
      'true',
    )
  },
}

/** Typing narrows the rows, the arrows move on them, and Enter chooses. */
export const Filtering: Story = {
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const field = canvas.getByRole('combobox')
    await userEvent.type(field, 'orchard')
    await waitFor(() => expect(canvas.getAllByRole('option')).toHaveLength(4))
    await userEvent.keyboard('{ArrowDown}{Enter}')
    await expect(args.onSelect).toHaveBeenCalledWith('default')
  },
}

/** Many rows: the row the arrow reaches is brought into view. */
export const LongList: Story = {
  args: { items: SHELVES, label: 'Shelves', placeholder: 'Search shelves' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const list = canvas.getByRole('listbox')
    await userEvent.type(canvas.getByRole('combobox'), '{ArrowUp}')
    await waitFor(() => expect(canvas.getByRole('option', { name: 'Shelf 60' })).toBeVisible())
    const active = canvas.getByRole('option', { name: 'Shelf 60' }).getBoundingClientRect()
    const inside = list.getBoundingClientRect()
    await expect(active.top).toBeLessThan(inside.bottom)
    await expect(active.bottom).toBeGreaterThan(inside.top)
    await expectOneFrame(canvasElement)
  },
}

/**
 * The item in force set from outside, as a reader's position moves on: the active row and the scroll
 * follow it, over a row the pointer rests on.
 */
export const FollowsTheItemInForce: Story = {
  render: () => ({
    components: { SearchableList },
    setup() {
      const current = ref<string | null>('shelf-0')
      return { current, items: SHELVES }
    },
    template: `
      <div class="numen" style="padding:2rem;inline-size:22rem;background:var(--numen-surface);color:var(--numen-ink)">
        <button type="button" @click="current = 'shelf-49'">Go to shelf 50</button>
        <SearchableList v-model="current" :items="items" label="Shelves" placeholder="Search shelves" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const list = canvas.getByRole('listbox')
    await userEvent.hover(canvas.getByRole('option', { name: 'Shelf 2' }))
    await waitFor(() =>
      expect(canvas.getByRole('option', { name: 'Shelf 2' })).toHaveAttribute(
        'aria-selected',
        'true',
      ),
    )

    await userEvent.click(canvas.getByRole('button', { name: 'Go to shelf 50' }))

    const target = canvas.getByRole('option', { name: 'Shelf 50' })
    await waitFor(() => expect(target).toHaveAttribute('aria-selected', 'true'))
    await waitFor(async () => {
      const box = target.getBoundingClientRect()
      const inside = list.getBoundingClientRect()
      await expect(box.top).toBeLessThan(inside.bottom)
      await expect(box.bottom).toBeGreaterThan(inside.top)
    })
  },
}

/** Nothing matches the words typed. */
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('combobox'), 'zzz')
    await expect(canvas.queryAllByRole('option')).toHaveLength(0)
    await expect(canvas.getByRole('status')).toHaveTextContent('No models found')
  },
}

/** The answer is awaited: the rows stay, and the frame says it is busy. */
export const Loading: Story = {
  args: { isPending: true, isFiltered: false },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getAllByRole('option')).toHaveLength(6)
    await expect(getFrame(canvasElement)).toHaveAttribute('aria-busy', 'true')
  },
}

/** Nothing is there yet and the answer is awaited. */
export const LoadingWithoutRows: Story = {
  args: { items: [], isPending: true, isFiltered: false },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('status')).toHaveTextContent('Searching')
  },
}

/** An action stands first while a name is typed, and Enter acts on it. */
export const WithAction: Story = {
  args: { items: SHELVES.slice(0, 3), isFiltered: false, hasAction: true, needsQuery: true },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await expect(canvas.queryAllByRole('option')).toHaveLength(0)
    await userEvent.type(canvas.getByRole('combobox'), 'Atlas')
    const options = await canvas.findAllByRole('option')
    await expect(options[0]).toHaveTextContent('Create "Atlas"')
    await expect(options[0]).toHaveAttribute('aria-selected', 'true')
    await userEvent.keyboard('{Enter}')
    await expect(args.onAct).toHaveBeenCalledWith('Atlas')
    await expectOneFrame(canvasElement)
  },
}

/** A row named exactly as typed takes the place of the action. */
export const ExactMatch: Story = {
  args: { items: SHELVES.slice(0, 3), isFiltered: false, hasAction: true, needsQuery: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('combobox'), 'shelf 2')
    const options = await canvas.findAllByRole('option')
    await expect(options[0]).toHaveTextContent('Shelf 2')
    await expect(canvas.queryByRole('option', { name: /^Create/ })).toBeNull()
  },
}

/** A popup at a point, drawn at the end of the document and kept inside the window. */
export const Popup: Story = {
  render: (args) => ({
    components: { SearchableListPopup },
    setup: () => ({ args }),
    template: `
      <SearchableListPopup
        :items="args.items"
        :label="args.label"
        :placeholder="args.placeholder"
        :at="{ x: 24, y: 24 }"
        :asking="240"
        @select="args.onSelect"
        @close="args.onClose"
      />
    `,
  }),
  play: async ({ args }) => {
    const frame = document.body.querySelector<HTMLElement>('[data-numen-searchable-list]')!
    await expect(frame.parentElement).toBe(document.body)
    const box = frame.getBoundingClientRect()
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
    await userEvent.keyboard('{Escape}')
    await expect(args.onClose).toHaveBeenCalled()
  },
}

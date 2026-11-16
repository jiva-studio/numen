/**
 * Stories of the searchable list in its multiple mode: rows with check marks, the keys that toggle
 * them, the row that creates an item from typed text, and a list of hundreds drawn by its visible
 * rows. Each story is also run as a test in a browser.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import SearchableList from './SearchableList.vue'
import type { ListItem } from './rows'
import { DARK, expectDark } from '@/shared/fixtures/theme'

interface Knobs {
  items: readonly ListItem[]
  recent: readonly string[]
  onToggle: (id: string) => void
  onAct: (query: string) => void
  onClose: () => void
}

const TREES: readonly ListItem[] = [
  { id: 'birch', text: 'Birch', state: 'unset' },
  { id: 'willow', text: 'Willow', state: 'unset' },
  { id: 'cedar', text: 'Cedar', state: 'mixed' },
  { id: 'alder', text: 'Alder', state: 'unset' },
  { id: 'maple', text: 'Maple', state: 'unset' },
  { id: 'fir', text: 'Fir', state: 'set' },
]

const HUNDREDS: readonly ListItem[] = Array.from({ length: 500 }, (_, index) => ({
  id: `grove-${index + 1}`,
  text: `Grove ${String(index + 1).padStart(3, '0')}`,
  state: 'unset',
}))

const meta: Meta<Knobs> = {
  title: 'Controls/SearchableListMultiple',
  parameters: { layout: 'centered' },
  args: {
    items: TREES,
    recent: ['maple'],
    onToggle: fn(),
    onAct: fn(),
    onClose: fn(),
  },
  render: (args) => ({
    components: { SearchableList },
    setup() {
      const items = ref<readonly ListItem[]>(args.items)
      const toggle = (id: string) => {
        items.value = items.value.map((one) =>
          one.id === id ? { ...one, state: one.state === 'set' ? 'unset' : 'set' } : one,
        )
        args.onToggle(id)
      }
      return { args, items, toggle }
    },
    template: `
      <div class="numen" style="padding:2rem;inline-size:22rem;background:var(--numen-surface);color:var(--numen-ink)">
        <SearchableList
          multiple
          has-action
          :items="items"
          :recent="args.recent"
          label="Trees"
          placeholder="Tree"
          @toggle="toggle"
          @act="args.onAct"
          @close="args.onClose"
        >
          <template #action="{ query }">Create {{ query }}</template>
        </SearchableList>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getRowNames = (canvas: ReturnType<typeof within>) =>
  canvas.queryAllByRole('option').map((one: HTMLElement) => one.textContent?.trim())

/** A set, a mixed and an unset item: a tick, a dash and an empty mark, the set ones first and the recent next. */
export const CheckMarks: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(getRowNames(canvas)).toStrictEqual([
      'Cedar',
      'Fir',
      'Maple',
      'Alder',
      'Birch',
      'Willow',
    ])
    await expect(canvas.getByRole('option', { name: 'Fir' })).toHaveAttribute(
      'aria-checked',
      'true',
    )
    await expect(canvas.getByRole('option', { name: 'Cedar' })).toHaveAttribute(
      'aria-checked',
      'mixed',
    )
    await expect(canvas.getByRole('option', { name: 'Birch' })).toHaveAttribute(
      'aria-checked',
      'false',
    )
  },
}

/** The same list in the dark token set. */
export const CheckMarksInTheDark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expectDark(canvasElement)
    await expect(canvas.getByRole('option', { name: 'Fir' })).toHaveAttribute(
      'aria-checked',
      'true',
    )
    await expect(canvas.getByRole('option', { name: 'Cedar' })).toHaveAttribute(
      'aria-checked',
      'mixed',
    )
    await expect(canvas.getByRole('option', { name: 'Birch' })).toHaveAttribute(
      'aria-checked',
      'false',
    )
  },
}

/** Enter and Control+K toggle the highlighted row, and the list stays open. */
export const TogglesWithEnterAndTheActionKey: Story = {
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('combobox'))
    await userEvent.keyboard('{ArrowDown}{ArrowDown}{Enter}')
    await expect(args.onToggle).toHaveBeenLastCalledWith('maple')
    await waitFor(() =>
      expect(canvas.getByRole('option', { name: 'Maple' })).toHaveAttribute('aria-checked', 'true'),
    )

    await userEvent.keyboard('{Control>}k{/Control}')
    await expect(args.onToggle).toHaveBeenCalledTimes(2)
    await waitFor(() =>
      expect(canvas.getByRole('option', { name: 'Maple' })).toHaveAttribute(
        'aria-checked',
        'false',
      ),
    )
    await expect(canvas.queryAllByRole('option')).toHaveLength(6)
    await expect(args.onClose).not.toHaveBeenCalled()
  },
}

/** Escape asks to close the list. */
export const ClosesOnEscape: Story = {
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('combobox'))
    await userEvent.keyboard('{Escape}')
    await expect(args.onClose).toHaveBeenCalledTimes(1)
  },
}

/** Typed text no item is named by makes a row that creates one; Enter asks for it and the list stays open. */
export const CreatesFromTypedText: Story = {
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('combobox'), 'Harbour')
    await waitFor(() => expect(getRowNames(canvas)).toStrictEqual(['Create Harbour']))
    await userEvent.keyboard('{Enter}')
    await expect(args.onAct).toHaveBeenCalledWith('Harbour')
    await expect(args.onClose).not.toHaveBeenCalled()
  },
}

/** Text that names an item, in another case, makes no create row and keeps the order of the rows. */
export const HidesTheCreateRowForAnExactTitle: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('combobox'), 'cedar')
    await waitFor(() => expect(getRowNames(canvas)).toStrictEqual(['Cedar']))
  },
}

/** Five hundred items: only the rows in sight are drawn, and the keyboard still reaches the last one. */
export const FarTooMany: Story = {
  args: { items: HUNDREDS, recent: [] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.queryAllByRole('option').length).toBeGreaterThan(0)
    await expect(canvas.queryAllByRole('option').length).toBeLessThan(100)
    await userEvent.click(canvas.getByRole('combobox'))
    await userEvent.keyboard('{End}')
    await waitFor(() => expect(canvas.getByRole('option', { name: 'Grove 500' })).toBeVisible())
    await expect(canvas.queryAllByRole('option').length).toBeLessThan(100)
    await expect(canvas.queryByRole('option', { name: 'Grove 001' })).toBeNull()
  },
}

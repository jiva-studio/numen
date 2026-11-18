/**
 * Stories of the palette asking for several items: the rows are the rows of the palette, with a dot
 * in the colour of the item before the name and a mark at the end of a row that is on. Each story
 * is also run as a test in a browser.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor } from 'storybook/test'
import { ref } from 'vue'
import Palette from './Palette.vue'

interface Item {
  readonly id: string
  readonly title: string
  readonly state: 'set' | 'mixed' | 'unset'
  readonly color: string
}

interface Knobs {
  items: readonly Item[]
  onToggle: (id: string) => void
  onCreate: (text: string) => void
  onDismiss: () => void
}

const TREES: readonly Item[] = [
  { id: 'birch', title: 'Birch', state: 'unset', color: 'rgb(200, 30, 40)' },
  { id: 'cedar', title: 'Cedar', state: 'mixed', color: 'rgb(30, 160, 60)' },
  { id: 'fir', title: 'Fir', state: 'set', color: 'rgb(40, 80, 220)' },
]

const meta = {
  title: 'Application/PaletteMultiple',
  parameters: { layout: 'fullscreen' },
  args: { items: TREES, onToggle: fn(), onCreate: fn(), onDismiss: fn() },
  render: (args: Knobs) => ({
    components: { Palette },
    setup() {
      const items = ref<readonly Item[]>(args.items)
      const toggle = (id: string) => {
        items.value = items.value.map((one) =>
          one.id === id ? { ...one, state: one.state === 'set' ? 'unset' : 'set' } : one,
        )
        args.onToggle(id)
      }
      const multi = () => ({
        title: 'Trees',
        placeholder: 'Tree',
        items: items.value,
        recent: [],
        getCreateTitle: (text: string) => `Create ${text}`,
        getCreateColor: () => 'rgb(120, 60, 200)',
      })
      return { args, multi, toggle }
    },
    template: `
      <div class="numen" style="height:100vh;background:var(--numen-surface);color:var(--numen-ink)">
        <Palette
          is-open
          :multi="multi()"
          @toggle="toggle"
          @create="args.onCreate"
          @dismiss="args.onDismiss"
        />
      </div>
    `,
  }),
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const options = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'))
const getName = (row: HTMLElement | undefined) => row?.textContent?.replace(/\s+/g, ' ').trim()
const getRow = (name: string) => options().find((one) => getName(one) === name)

const isPaintedWith = (row: HTMLElement | undefined, color: string): boolean =>
  row?.querySelector('[data-palette="icon"]')?.outerHTML.includes(color) === true

/** The icon of each row is a dot in the colour of its item, and a row that is on shows its mark at the end. */
export const DotsAndMarks: Story = {
  play: async () => {
    await waitFor(() => expect(options()).toHaveLength(3))
    await expect(options().map(getName)).toStrictEqual(['Cedar', 'Fir', 'Birch'])
    await expect(isPaintedWith(getRow('Birch'), 'rgb(200, 30, 40)')).toBe(true)
    await expect(isPaintedWith(getRow('Cedar'), 'rgb(30, 160, 60)')).toBe(true)
    await expect(isPaintedWith(getRow('Fir'), 'rgb(40, 80, 220)')).toBe(true)
    await expect(getRow('Fir')?.lastElementChild).toHaveAttribute('data-palette', 'check')
    await expect(getRow('Cedar')?.lastElementChild).toHaveAttribute('data-state', 'mixed')
    await expect(getRow('Birch')?.querySelector('[data-palette="check"]')).toBeNull()
  },
}

/** Enter toggles the highlighted row and the palette stays open; the mark follows. */
export const TogglesWithEnter: Story = {
  play: async ({ args }) => {
    await waitFor(() => expect(options()).toHaveLength(3))
    await userEvent.click(document.body.querySelector<HTMLElement>('[data-palette="field"]')!)
    await userEvent.keyboard('{ArrowDown}{ArrowDown}{Enter}')
    await expect(args.onToggle).toHaveBeenLastCalledWith('birch')
    await waitFor(() =>
      expect(getRow('Birch')?.lastElementChild).toHaveAttribute('data-palette', 'check'),
    )
    await expect(args.onDismiss).not.toHaveBeenCalled()
  },
}

/** Typed text no item is named by makes a row with the colour the caller gives; Enter asks to create. */
export const CreatesFromTypedText: Story = {
  play: async ({ args }) => {
    await userEvent.type(
      document.body.querySelector<HTMLElement>('[data-palette="field"]')!,
      'Harbour',
    )
    await waitFor(() => expect(options().map(getName)).toStrictEqual(['Create Harbour']))
    await expect(isPaintedWith(getRow('Create Harbour'), 'rgb(120, 60, 200)')).toBe(true)
    await userEvent.keyboard('{Enter}')
    await expect(args.onCreate).toHaveBeenCalledWith('Harbour')
    await expect(args.onDismiss).not.toHaveBeenCalled()
  },
}

/** Escape asks to close the palette. */
export const ClosesOnEscape: Story = {
  play: async ({ args }) => {
    await userEvent.click(document.body.querySelector<HTMLElement>('[data-palette="field"]')!)
    await userEvent.keyboard('{Escape}')
    await expect(args.onDismiss).toHaveBeenCalledTimes(1)
  },
}

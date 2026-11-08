/**
 * The combobox a thought is named in when a link is dragged out into empty space.
 *
 * The input and the options are one frame, the kit's searchable list; its geometry is the
 * browser's answer.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { DARK, expectDark } from '../fixtures/theme'
import PlexQuickLinkPopover from './PlexQuickLinkPopover.vue'

type Found = { path: string; title: string }[]

const BOOKS: Found = [
  { path: 'notes/books.md', title: 'Books' },
  { path: 'notes/bhagavad-gita.md', title: 'Bhagavad-gita' },
]

const LONG: Found = Array.from({ length: 30 }, (_, index) => ({
  path: `notes/shelf-${index}.md`,
  title: `Shelf ${index}`,
}))

const meta = {
  title: 'Plex/Quick Link Popover',
  component: PlexQuickLinkPopover,
  parameters: { layout: 'fullscreen' },
  args: { at: { x: 300.4, y: 200.6 }, relation: 'child', search: async () => [] },
} satisfies Meta<typeof PlexQuickLinkPopover>

export default meta
type Story = StoryObj<typeof meta>

const getFrame = (canvasElement: HTMLElement) =>
  canvasElement.querySelector<HTMLElement>('[data-numen-searchable-list]')!

const expectOneFrame = async (canvasElement: HTMLElement, hasList: boolean) => {
  const canvas = within(canvasElement)
  const frame = getFrame(canvasElement)
  await Promise.all(frame.getAnimations({ subtree: true }).map((animation) => animation.finished))
  const field = canvas.getByRole('combobox')
  await expect(frame.contains(field)).toBe(true)

  await expect(frame.firstElementChild).toBe(field)
  await expect(getComputedStyle(frame).overflow).toBe('hidden')

  if (!hasList) return
  const list = canvas.getByRole('listbox')
  await expect(frame.contains(list)).toBe(true)
  const outer = frame.getBoundingClientRect()
  const inner = list.getBoundingClientRect()
  const row = field.getBoundingClientRect()
  await expect(inner.top).toBeGreaterThanOrEqual(row.bottom)
  await expect(inner.top).toBeLessThanOrEqual(row.bottom + 2)
  await expect(inner.left).toBeGreaterThanOrEqual(outer.left)
  await expect(inner.right).toBeLessThanOrEqual(outer.right)
  await expect(inner.bottom).toBeLessThanOrEqual(outer.bottom)
}

const run: Story['play'] = async ({ canvasElement }) => {
  const canvas = within(canvasElement)
  await userEvent.type(canvas.getByRole('combobox'), 'B')
  await waitFor(() => canvas.getByRole('option', { name: 'Create "B"' }))
}

/** Nothing typed: the frame is the input alone, with no list under it. */
export const Empty: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.queryByRole('listbox')).toBeNull()
    await expectOneFrame(canvasElement, false)
  },
}

/** The search has not answered: Create is the one row, and it is active. */
export const Loading: Story = {
  args: { search: () => new Promise<never>(() => undefined) },
  play: async (context) => {
    await run(context)
    const canvas = within(context.canvasElement)
    await expect(canvas.getAllByRole('option')).toHaveLength(1)
    await expect(canvas.getByRole('option', { name: 'Create "B"' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    await expectOneFrame(context.canvasElement, true)
  },
}

/** Found notes follow Create, and Create stays the active row. */
export const Results: Story = {
  args: { search: async () => BOOKS },
  play: async (context) => {
    await run(context)
    const canvas = within(context.canvasElement)
    await waitFor(() => expect(canvas.getAllByRole('option')).toHaveLength(3))
    await expect(canvas.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'Create "B"',
      'Books',
      'Bhagavad-gita',
    ])
    await expect(canvas.getByRole('option', { name: 'Create "B"' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    await expectOneFrame(context.canvasElement, true)
  },
}

/** A note has exactly the typed title: it takes row 1 and Create is not offered. */
export const ExactMatch: Story = {
  args: { search: async () => BOOKS },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('combobox'), 'books')
    await waitFor(() => expect(canvas.getAllByRole('option')[0]).toHaveTextContent('Books'))
    await expect(canvas.queryByRole('option', { name: /^Create/ })).toBeNull()
    await expect(canvas.getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true')
    await expectOneFrame(canvasElement, true)
  },
}

/** Nothing matches, so the one row offered is the creation of the name typed. */
export const NoResults: Story = {
  play: async (context) => {
    await run(context)
    const canvas = within(context.canvasElement)
    await expect(canvas.getAllByRole('option')).toHaveLength(1)
    await expectOneFrame(context.canvasElement, true)
  },
}

/** Many notes: the list scrolls inside the frame and the frame stops growing. */
export const LongList: Story = {
  args: { search: async () => LONG },
  play: async (context) => {
    await run(context)
    const canvas = within(context.canvasElement)
    await waitFor(() => expect(canvas.getAllByRole('option')).toHaveLength(31))
    const list = canvas.getByRole('listbox')
    await expect(list.scrollHeight).toBeGreaterThan(list.clientHeight)
    await expect(getComputedStyle(list).overflowY).toBe('auto')
    await expectOneFrame(context.canvasElement, true)
  },
}

/** The results in the dark token set. */
export const ResultsDark: Story = {
  ...Results,
  globals: DARK,
  play: async (context) => {
    await expectDark(context.canvasElement)
    await Results.play!(context)
  },
}

/** An empty frame in the dark token set. */
export const EmptyDark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await expectOneFrame(canvasElement, false)
  },
}

/** A long list in the dark token set. */
export const LongListDark: Story = {
  ...LongList,
  globals: DARK,
  play: async (context) => {
    await expectDark(context.canvasElement)
    await LongList.play!(context)
  },
}

/** The pending search in the dark token set. */
export const LoadingDark: Story = {
  ...Loading,
  globals: DARK,
  play: async (context) => {
    await expectDark(context.canvasElement)
    await Loading.play!(context)
  },
}

/** The creation row alone in the dark token set. */
export const NoResultsDark: Story = {
  ...NoResults,
  globals: DARK,
  play: async (context) => {
    await expectDark(context.canvasElement)
    await NoResults.play!(context)
  },
}

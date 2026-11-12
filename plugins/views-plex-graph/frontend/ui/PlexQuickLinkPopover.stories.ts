/**
 * The combobox a thought is named in when a link is dragged out into empty space.
 *
 * The input and the options are one frame, the kit's searchable list.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
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

/** Releases the search of the story that holds its answer. */
let answerSearch: (() => void) | null = null

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
  const field = canvas.getByRole('combobox')
  await expect(frame.contains(field)).toBe(true)

  await expect(frame.firstElementChild).toBe(field)

  if (!hasList) return
  await expect(frame.contains(canvas.getByRole('listbox'))).toBe(true)
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

/**
 * The exact match arrives after the search answers: row 1 changes from Create to the note and
 * keeps its place and size, and Enter links the note without creating one.
 */
export const ExactMatchArrivesLate: Story = {
  args: {
    search: () =>
      new Promise((resolve) => {
        answerSearch = () => resolve(BOOKS)
      }),
    'onSelect-note': fn(),
    'onCreate-note': fn(),
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    answerSearch = null
    const field = canvas.getByRole('combobox')
    await userEvent.type(field, 'books')
    const first = await canvas.findByRole('option', { name: 'Create "books"' })
    const before = first.getBoundingClientRect()

    await waitFor(() => expect(answerSearch).not.toBeNull())
    answerSearch!()

    await waitFor(() => expect(canvas.getAllByRole('option')[0]).toHaveTextContent('Books'))
    await expect(canvas.queryByRole('option', { name: /^Create/ })).toBeNull()
    const row = canvas.getAllByRole('option')[0]!
    await expect(row).toHaveAttribute('aria-selected', 'true')
    const after = row.getBoundingClientRect()
    await expect([after.top, after.left, after.width, after.height]).toEqual([
      before.top,
      before.left,
      before.width,
      before.height,
    ])

    await userEvent.keyboard('{Enter}')
    await expect(args['onSelect-note']).toHaveBeenCalledWith('notes/books.md')
    await expect(args['onCreate-note']).not.toHaveBeenCalled()
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

/** Many notes: every one is offered in the one frame. */
export const LongList: Story = {
  args: { search: async () => LONG },
  play: async (context) => {
    await run(context)
    const canvas = within(context.canvasElement)
    await waitFor(() => expect(canvas.getAllByRole('option')).toHaveLength(31))
    await expectOneFrame(context.canvasElement, true)
  },
}

/**
 * The title bar item that counts the vectors of the search index, with its popover open. Also the
 * test corpus: each story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import type { Stats } from '../types'
import StatsIndicator from './StatsIndicator.vue'
import '../styles.css'

/** A made-up vault that is embedding: notes, a PDF and a book, with no media in it. */
const EMBEDDING: Stats = {
  vectors: 12480,
  chunks: 13100,
  waiting: 620,
  sources: 84,
  kinds: [
    { kind: 'notes', sources: 70, chunks: 4100 },
    { kind: 'pdf', sources: 9, chunks: 7000 },
    { kind: 'epub', sources: 5, chunks: 2000 },
    { kind: 'media', sources: 0, chunks: 0 },
  ],
  model: 'bge-m3',
  recipe: 'local|bge-m3|1024|512|mean|int8@0.4',
  dimensions: 1024,
  profile: 'balanced',
  threads: 4,
  speed: 18.5,
  changedAt: '2026-11-02T12:00:07Z',
  databaseBytes: 73400320,
  vectorBytes: 52428800,
}

const { speed: _speed, ...MEASURED } = EMBEDDING

/** The same vault once every chunk carries a vector. */
const DONE: Stats = { ...MEASURED, vectors: 13100, waiting: 0 }

const meta = {
  title: 'Index stats/Indicator',
  component: StatsIndicator,
  parameters: { layout: 'fullscreen' },
  args: { stats: EMBEDDING, phase: 'ready', locale: 'en-US', onOpen: fn(), onClose: fn() },
  render: (args) => ({
    components: { StatsIndicator },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:flex-start;justify-content:flex-end;block-size:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <header style="display:flex;align-items:center;justify-content:flex-end;position:relative;inline-size:100%;block-size:2.25rem;background:var(--numen-surface);border-bottom:1px solid var(--numen-rule);font-size:var(--numen-text-1)">
          <StatsIndicator v-bind="args" />
        </header>
      </div>
    `,
  }),
} satisfies Meta<typeof StatsIndicator>

export default meta
type Story = StoryObj<typeof meta>

/** The item over the vectors and the chunks that should carry one, and its popover with one row per kind that holds sources. */
export const Open: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /12,480/ }))
    const popover = await canvas.findByRole('dialog', { name: 'Index stats' })
    for (const label of ['Vectors', 'Chunks', 'Waiting for a vector', 'Notes', 'PDF', 'EPUB']) {
      await expect(within(popover).getByText(label)).toBeVisible()
    }
    await expect(within(popover).queryByText('Media transcripts')).toBeNull()
  },
}

/** Once the index is embedded the item shows the vectors alone and the popover has no speed row. */
export const Embedded: Story = {
  args: { stats: DONE },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = await canvas.findByRole('button', { name: /13,100/ })
    await expect(trigger.textContent?.trim()).toBe('⬡13,100')
    await userEvent.click(trigger)
    const popover = await canvas.findByRole('dialog')
    await expect(within(popover).queryByText('Embedding speed')).toBeNull()
  },
}

/** While chunks wait the item names the vectors and says it is indexing. */
export const Indexing: Story = {
  play: async ({ canvasElement }) => {
    const trigger = await within(canvasElement).findByRole('button', {
      name: '12,480 vectors, indexing',
    })
    await expect(trigger.textContent?.trim()).toBe('⬡12,480')
    await expect(trigger.textContent).not.toContain('/')
  },
}

/** Once the queue is empty the item names the vectors alone. */
export const Idle: Story = {
  args: { stats: DONE },
  play: async ({ canvasElement }) => {
    const trigger = await within(canvasElement).findByRole('button', { name: '13,100 vectors' })
    await expect(trigger.textContent?.trim()).toBe('⬡13,100')
  },
}

/** The item says when its popover opens and closes, which is when the window asks for fresh counts. */
export const OpenAndClose: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = await canvas.findByRole('button', { name: /12,480/ })
    await userEvent.click(trigger)
    await expect(args.onOpen).toHaveBeenCalledTimes(1)
    await expect(args.onClose).not.toHaveBeenCalled()
    const popover = await canvas.findByRole('dialog')
    await userEvent.click(within(popover).getByRole('button', { name: 'Close index stats' }))
    await expect(args.onClose).toHaveBeenCalledTimes(1)
  },
}

/**
 * The status bar item that counts the vectors of the search index, with its popover open. Also the
 * test corpus: each story is run in a browser by `@storybook/addon-vitest`.
 *
 * The popover is drawn above the item and must not be clipped by the bar or the window, and its
 * ground must be a colour of the theme and never see-through; only a browser lays that out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import type { Stats } from './types'
import StatsIndicator from './StatsIndicator.vue'
import './styles.css'

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
  args: { stats: EMBEDDING, phase: 'ready', locale: 'en-US' },
  render: (args) => ({
    components: { StatsIndicator },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:flex-end;justify-content:flex-end;block-size:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <footer style="display:flex;align-items:center;justify-content:flex-end;position:relative;inline-size:100%;block-size:1.375rem;background:var(--numen-surface);border-top:1px solid var(--numen-rule);font-size:var(--numen-text-1)">
          <StatsIndicator v-bind="args" />
        </footer>
      </div>
    `,
  }),
} satisfies Meta<typeof StatsIndicator>

export default meta
type Story = StoryObj<typeof meta>

const getShade = (colour: string): { shade: number; alpha: number } => {
  const channels = (colour.match(/[\d.]+/g) ?? []).map(Number)
  const [red = 0, green = 0, blue = 0, alpha = 1] = channels
  return { shade: red + green + blue, alpha }
}

const openAndMeasure: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const canvas = within(canvasElement)
  await userEvent.click(await canvas.findByRole('button', { name: /12,480/ }))
  const popover = await canvas.findByRole('dialog', { name: 'Index stats' })
  const bar = canvasElement.querySelector('footer') as HTMLElement

  await waitFor(() => {
    const box = popover.getBoundingClientRect()
    return expect(box.bottom).toBeLessThanOrEqual(bar.getBoundingClientRect().top)
  })
  await expect(popover.getBoundingClientRect().width).toBeGreaterThan(200)
  const box = popover.getBoundingClientRect()
  await expect(box.top).toBeGreaterThanOrEqual(0)
  await expect(box.left).toBeGreaterThanOrEqual(0)
  await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
  await expect(getShade(getComputedStyle(popover).backgroundColor).alpha).toBe(1)

  for (const label of ['Vectors', 'Chunks', 'Waiting for a vector', 'Notes', 'PDF', 'EPUB']) {
    await expect(within(popover).getByText(label)).toBeVisible()
  }
  await expect(within(popover).queryByText('Media transcripts')).toBeNull()
}

/** The item over the vectors and the chunks that should carry one, and its popover, on the light tokens. */
export const Light: Story = {
  globals: { theme: 'light' },
  play: async (context) => {
    await openAndMeasure(context)
    const popover = within(context.canvasElement).getByRole('dialog')
    await expect(getShade(getComputedStyle(popover).backgroundColor).shade).toBeGreaterThan(384)
  },
}

/** The same on the dark tokens: the ground is dark and the text is read against it. */
export const Dark: Story = {
  globals: { theme: 'dark' },
  play: async (context) => {
    await openAndMeasure(context)
    const root = context.canvasElement.ownerDocument.documentElement
    await expect(getComputedStyle(root).colorScheme).toBe('dark')
    const popover = within(context.canvasElement).getByRole('dialog')
    await expect(getShade(getComputedStyle(popover).backgroundColor).shade).toBeLessThan(384)
    await expect(getShade(getComputedStyle(popover).color).shade).toBeGreaterThan(384)
  },
}

/** Once the index is embedded the item shows the vectors alone and the popover has no speed row. */
export const Embedded: Story = {
  args: { stats: DONE },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = await canvas.findByRole('button', { name: /13,100/ })
    await expect(trigger.textContent?.trim()).toBe('⬡ 13,100')
    await userEvent.click(trigger)
    const popover = await canvas.findByRole('dialog')
    await expect(within(popover).queryByText('Embedding speed')).toBeNull()
  },
}

const getIcon = (canvasElement: HTMLElement): HTMLElement =>
  canvasElement.querySelector('.index-stats__icon') as HTMLElement

const checkIndexing: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const canvas = within(canvasElement)
  const trigger = await canvas.findByRole('button', { name: '12,480 vectors, indexing' })
  await expect(trigger.textContent?.trim()).toBe('⬡ 12,480')
  await expect(trigger.textContent).not.toContain('/')
  const icon = getIcon(canvasElement)
  await expect(getComputedStyle(icon).animationName).not.toBe('none')
  await expect(icon.getAnimations().some((one) => one.playState === 'running')).toBe(true)
  const label = canvasElement.querySelector('.index-stats__label') as HTMLElement
  await expect(getComputedStyle(icon).color).not.toBe(getComputedStyle(label).color)
  await expect(getComputedStyle(label).animationName).toBe('none')
}

const checkIdle: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const canvas = within(canvasElement)
  const trigger = await canvas.findByRole('button', { name: '13,100 vectors' })
  await expect(trigger.textContent?.trim()).toBe('⬡ 13,100')
  const icon = getIcon(canvasElement)
  await expect(getComputedStyle(icon).animationName).toBe('none')
  await expect(icon.getAnimations()).toHaveLength(0)
  const label = canvasElement.querySelector('.index-stats__label') as HTMLElement
  await expect(getComputedStyle(icon).color).toBe(getComputedStyle(label).color)
}

/** The mark pulses green on the light tokens while chunks wait. */
export const IndexingLight: Story = { globals: { theme: 'light' }, play: checkIndexing }

/** The same on the dark tokens. */
export const IndexingDark: Story = { globals: { theme: 'dark' }, play: checkIndexing }

/** The mark is still on the light tokens once the queue is empty. */
export const IdleLight: Story = {
  args: { stats: DONE },
  globals: { theme: 'light' },
  play: checkIdle,
}

/** The same on the dark tokens. */
export const IdleDark: Story = {
  args: { stats: DONE },
  globals: { theme: 'dark' },
  play: checkIdle,
}

/** With no motion in the theme the mark stays solid green and does not fade. */
export const IndexingWithoutMotion: Story = {
  decorators: [
    (story) => ({
      components: { story },
      template: '<div style="--numen-motion: 0ms"><story /></div>',
    }),
  ],
  play: async ({ canvasElement }) => {
    const icon = getIcon(canvasElement)
    await expect(icon).toBeVisible()
    await expect(getComputedStyle(icon).animationDuration).toBe('0s')
    await expect(getComputedStyle(icon).opacity).toBe('1')
    await expect(getComputedStyle(icon).color).not.toBe(
      getComputedStyle(canvasElement.querySelector('.index-stats__label') as HTMLElement).color,
    )
  },
}

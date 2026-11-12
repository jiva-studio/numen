/**
 * The title bar item that counts the vectors of the search index, with its popover open. Also the
 * test corpus: each story is run in a browser by `@storybook/addon-vitest`.
 *
 * The popover is drawn below the item and must not be clipped by the bar or the window, and its
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

const getShade = (colour: string): { shade: number; alpha: number } => {
  const channels = (colour.match(/[\d.]+/g) ?? []).map(Number)
  const [red = 0, green = 0, blue = 0, alpha = 1] = channels
  return { shade: red + green + blue, alpha }
}

const openAndMeasure: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const canvas = within(canvasElement)
  await userEvent.click(await canvas.findByRole('button', { name: /12,480/ }))
  const popover = await within(document.body).findByRole('dialog', { name: 'Index stats' })
  const bar = canvasElement.querySelector('header') as HTMLElement

  await waitFor(() => {
    const box = popover.getBoundingClientRect()
    return expect(box.top).toBeGreaterThanOrEqual(bar.getBoundingClientRect().bottom)
  })
  await expect(popover.getBoundingClientRect().width).toBeGreaterThan(200)
  const box = popover.getBoundingClientRect()
  await expect(box.bottom).toBeLessThanOrEqual(window.innerHeight)
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
    const popover = within(document.body).getByRole('dialog')
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
    const popover = within(document.body).getByRole('dialog')
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
    await expect(trigger.textContent?.trim()).toBe('⬡13,100')
    await userEvent.click(trigger)
    const popover = await within(document.body).findByRole('dialog')
    await expect(within(popover).queryByText('Embedding speed')).toBeNull()
  },
}

const getBox = (element: HTMLElement) => {
  const { width, height } = element.getBoundingClientRect()
  return { width, height }
}

const checkIndexing: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const canvas = within(canvasElement)
  const trigger = await canvas.findByRole('button', { name: '12,480 vectors, indexing' })
  await expect(trigger.textContent?.trim()).toBe('⬡12,480')
  await expect(trigger.textContent).not.toContain('/')
  await expect(trigger.getAnimations()).toHaveLength(0)
  await expect(getComputedStyle(trigger).borderTopWidth).toBe('0px')
}

const checkIdle: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const canvas = within(canvasElement)
  const trigger = await canvas.findByRole('button', { name: '13,100 vectors' })
  await expect(trigger.textContent?.trim()).toBe('⬡13,100')
  await expect(getComputedStyle(trigger).animationName).toBe('none')
  await expect(trigger.getAnimations()).toHaveLength(0)
}

/** The chip is still while chunks wait, on the light tokens. */
export const IndexingLight: Story = { globals: { theme: 'light' }, play: checkIndexing }

/** The same on the dark tokens. */
export const IndexingDark: Story = { globals: { theme: 'dark' }, play: checkIndexing }

/** The chip is still on the light tokens once the queue is empty. */
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

/** The chip is 20px high whether it is indexing, idle, loading or without an index. */
export const KeepsItsSize: Story = {
  render: () => ({
    components: { StatsIndicator },
    setup: () => ({ EMBEDDING, DONE }),
    template: `
      <div class="numen" data-chips style="display:flex;align-items:center;gap:8px;block-size:2rem;background:var(--numen-titlebar);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <StatsIndicator :stats="EMBEDDING" phase="ready" locale="en-US" />
        <StatsIndicator :stats="DONE" phase="ready" locale="en-US" />
        <StatsIndicator :stats="null" phase="loading" locale="en-US" />
        <StatsIndicator :stats="null" phase="unavailable" locale="en-US" />
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const boxes = within(canvasElement).getAllByRole('button').map(getBox)
    await expect(boxes).toHaveLength(4)
    for (const box of boxes) await expect(box.height).toBe(20)
  },
}

const inBar = (place: 'top' | 'bottom') => () => ({
  components: { StatsIndicator },
  setup: () => ({ EMBEDDING }),
  template: `
    <div class="numen" style="block-size: 100vh; background: var(--numen-surface); color: var(--numen-ink); font-family: var(--numen-font-sans)">
      <div data-view style="position: fixed; ${place}: 6px; right: 320px; display: inline-flex; align-items: center">
        <StatsIndicator :stats="EMBEDDING" phase="ready" locale="en-US" />
      </div>
    </div>`,
})

const getChip = (canvasElement: HTMLElement): HTMLElement =>
  within(canvasElement.querySelector<HTMLElement>('[data-view]')!).getAllByRole('button')[0]!

/** The close button of the dialog closes it, and a resize of the window keeps it open. */
export const ClosesFromItsButtonAndStaysOpenOnAResize: Story = {
  render: inBar('top'),
  play: async ({ canvasElement }) => {
    await userEvent.click(getChip(canvasElement))
    const dialog = await within(document.body).findByRole('dialog', { name: 'Index stats' })
    window.dispatchEvent(new Event('resize'))
    await new Promise((resolve) => requestAnimationFrame(resolve))
    await expect(within(document.body).queryByRole('dialog')).not.toBeNull()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Close index stats' }))
    await waitFor(() => expect(within(document.body).queryByRole('dialog')).toBeNull())
    await expect(getChip(canvasElement).getAttribute('aria-expanded')).toBe('false')
  },
}

/** In a bar at the bottom of the window the dialog opens above the chip, inside the window. */
export const OpensTheDialogAboveAChipAtTheBottom: Story = {
  render: inBar('bottom'),
  play: async ({ canvasElement }) => {
    const chip = getChip(canvasElement)
    await userEvent.click(chip)
    const dialog = await within(document.body).findByRole('dialog', { name: 'Index stats' })
    await waitFor(() =>
      expect(dialog.getBoundingClientRect().bottom).toBeLessThanOrEqual(
        chip.getBoundingClientRect().top,
      ),
    )
    await expect(dialog.getBoundingClientRect().top).toBeGreaterThanOrEqual(0)
  },
}

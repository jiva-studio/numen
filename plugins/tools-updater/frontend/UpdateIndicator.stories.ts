/**
 * The version chip of the title bar while an update waits, in the accent. Also the test corpus: each story is run in a browser by
 * `@storybook/addon-vitest`, which reads the size and the contrast of the chip as drawn.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import type { Update } from './api/updater'
import UpdateIndicator from './UpdateIndicator.vue'
import './styles.css'

const UPDATE: Update = {
  version: '2026.11.12',
  build: 960,
  channel: 'stable',
  releasedAt: '2026-11-08T12:00:00Z',
  notesUrl: 'https://example.org/notes',
  downloadUrl: 'https://example.org/download',
}

const meta = {
  title: 'Update notifier/Version chip',
  component: UpdateIndicator,
  parameters: { layout: 'fullscreen' },
  args: { version: '2026.11.2', update: UPDATE, platform: 'linux' },
  render: (args) => ({
    components: { UpdateIndicator },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:center;justify-content:flex-end;block-size:2rem;background:var(--numen-titlebar);border-bottom:1px solid var(--numen-rule);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <UpdateIndicator v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof UpdateIndicator>

export default meta
type Story = StoryObj<typeof meta>

const checkAvailable: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = within(canvasElement).getByRole('button')
  await expect(chip.textContent?.trim()).toBe('v2026.11.12')
  await expect(chip.getAttribute('title')).toContain('Update available')
  await expect(getComputedStyle(chip).backgroundColor).not.toBe(
    getComputedStyle(canvasElement.querySelector('div') as HTMLElement).backgroundColor,
  )
}

/** An update waits: the chip takes the accent, on the light tokens. */
export const AvailableLight: Story = {
  globals: { theme: 'light' },
  play: checkAvailable,
}

/** The same on the dark tokens. */
export const AvailableDark: Story = {
  globals: { theme: 'dark' },
  play: checkAvailable,
}

const inBar = (place: 'top' | 'bottom') => () => ({
  components: { UpdateIndicator },
  setup: () => ({ UPDATE }),
  template: `
    <div class="numen" style="block-size: 100vh; background: var(--numen-surface); color: var(--numen-ink); font-family: var(--numen-font-sans)">
      <div data-view style="position: fixed; ${place}: 6px; right: 320px; display: inline-flex; align-items: center">
        <UpdateIndicator version="2026.11.2" :update="UPDATE" platform="linux" />
      </div>
    </div>`,
})

const getChip = (canvasElement: HTMLElement): HTMLElement =>
  within(canvasElement.querySelector<HTMLElement>('[data-view]')!).getAllByRole('button')[0]!

const openRelease = async (canvasElement: HTMLElement): Promise<HTMLElement> => {
  await userEvent.click(getChip(canvasElement))
  return within(document.body).findByRole('dialog', { name: 'Update available' })
}

/** The chip is 20px high. */
export const KeepsItsSize: Story = {
  render: () => ({
    components: { UpdateIndicator },
    setup: () => ({ UPDATE }),
    template: `
      <div class="numen" style="display:flex;align-items:center;gap:8px;block-size:2rem;background:var(--numen-titlebar);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <UpdateIndicator version="2026.11.2" :update="UPDATE" platform="linux" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const boxes = within(canvasElement)
      .getAllByRole('button')
      .map((one) => one.getBoundingClientRect())
    await expect(boxes).toHaveLength(1)
    for (const box of boxes) await expect(box.height).toBe(20)
  },
}

/** A press opens the release dialog; each of its actions closes it. */
export const ClosesTheReleaseDialogFromEachAction: Story = {
  render: inBar('top'),
  play: async ({ canvasElement }) => {
    for (const action of ['Download', 'Release notes', 'Dismiss']) {
      const dialog = await openRelease(canvasElement)
      await expect(getChip(canvasElement).getAttribute('aria-expanded')).toBe('true')
      await userEvent.click(within(dialog).getByRole('button', { name: action }))
      await waitFor(() => expect(within(document.body).queryByRole('dialog')).toBeNull())
      await expect(getChip(canvasElement).getAttribute('aria-expanded')).toBe('false')
    }
  },
}

/** In a bar at the bottom of the window the release dialog opens above the chip, inside the window. */
export const OpensTheReleaseDialogAboveAChipAtTheBottom: Story = {
  render: inBar('bottom'),
  play: async ({ canvasElement }) => {
    const dialog = await openRelease(canvasElement)
    const chip = getChip(canvasElement)
    await waitFor(() =>
      expect(dialog.getBoundingClientRect().bottom).toBeLessThanOrEqual(
        chip.getBoundingClientRect().top,
      ),
    )
    await expect(dialog.getBoundingClientRect().top).toBeGreaterThanOrEqual(0)
  },
}

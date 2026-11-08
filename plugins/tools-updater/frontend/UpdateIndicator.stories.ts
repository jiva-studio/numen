/**
 * The version chip of the title bar in its states: the version that runs, neutral, and an update
 * that waits, in the accent. Also the test corpus: each story is run in a browser by
 * `@storybook/addon-vitest`, which reads the size and the contrast of the chip as drawn.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import type { Update } from './api/updater'
import UpdateIndicator from './UpdateIndicator.vue'
import { expectText } from '../../../libs/ui/core/.storybook/contrastBrowser'
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
  args: { version: '2026.11.2', state: 'current', update: null, platform: 'linux' },
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

const getChip = (canvasElement: HTMLElement): HTMLElement =>
  canvasElement.querySelector('.update-indicator__trigger') as HTMLElement

const checkCurrent: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = getChip(canvasElement)
  await expect(chip.textContent?.trim()).toBe('v2026.11.2')
  await expect(chip.getAttribute('title')).toContain('latest version')
  await expect(chip.tagName).toBe('SPAN')
  await expectText(chip, 'version chip', 'current')
}

const checkAvailable: NonNullable<Story['play']> = async ({ canvasElement }) => {
  const chip = within(canvasElement).getByRole('button')
  await expect(chip.textContent?.trim()).toBe('v2026.11.12')
  await expect(chip.getAttribute('title')).toContain('Update available')
  await expect(getComputedStyle(chip).backgroundColor).not.toBe(
    getComputedStyle(canvasElement.querySelector('div') as HTMLElement).backgroundColor,
  )
  await expectText(chip, 'version chip', 'available')
}

/** The version that runs, neutral, on the light tokens. */
export const CurrentLight: Story = { globals: { theme: 'light' }, play: checkCurrent }

/** The same on the dark tokens. */
export const CurrentDark: Story = { globals: { theme: 'dark' }, play: checkCurrent }

/** An update waits: the chip takes the accent, on the light tokens. */
export const AvailableLight: Story = {
  args: { state: 'available', update: UPDATE },
  globals: { theme: 'light' },
  play: checkAvailable,
}

/** The same on the dark tokens. */
export const AvailableDark: Story = {
  args: { state: 'available', update: UPDATE },
  globals: { theme: 'dark' },
  play: checkAvailable,
}

/** The chip keeps its height and its width whether it shows the running version or the update. */
export const KeepsItsSize: Story = {
  render: () => ({
    components: { UpdateIndicator },
    setup: () => ({ UPDATE }),
    template: `
      <div class="numen" style="display:flex;align-items:center;gap:8px;block-size:2rem;background:var(--numen-titlebar);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <UpdateIndicator version="2026.11.2" state="current" :update="null" platform="linux" />
        <UpdateIndicator version="2026.11.2" state="available" :update="UPDATE" platform="linux" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const boxes = [...canvasElement.querySelectorAll<HTMLElement>('.update-indicator__trigger')].map(
      (one) => {
        const { width, height } = one.getBoundingClientRect()
        return { width, height }
      },
    )
    await expect(boxes).toHaveLength(2)
    await expect(boxes[1]).toStrictEqual(boxes[0])
  },
}

/**
 * The version chip of the title bar while an update waits. Also the test corpus: each story is run in a browser by
 * `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import type { Update } from '../api/updater'
import UpdateIndicator from './UpdateIndicator.vue'
import '../styles.css'

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

/** An update waits: the chip names the new version and says an update is available. */
export const Available: Story = {
  play: async ({ canvasElement }) => {
    const chip = within(canvasElement).getByRole('button')
    await expect(chip.textContent?.trim()).toBe('v2026.11.12')
    await expect(chip.getAttribute('title')).toContain('Update available')
  },
}

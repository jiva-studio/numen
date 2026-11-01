/**
 * The plugin list of the settings tab under the default theme, rendered in a browser.
 *
 * A story checks the computed corners: an undeclared radius token draws a square corner in
 * every engine without any error.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { ref } from 'vue'
import { Button } from '@numen/ui'
import PluginsList from './PluginsList.vue'
import type { PluginInfo } from '@/entities/settings'
import { createInstallation, createPluginInfo } from '@/testing/installation'

const PLUGINS: readonly PluginInfo[] = [
  createPluginInfo('sample.notes', 'Sample Notes', { isActive: true }),
  createPluginInfo('sample.tree', 'Sample Tree'),
]

const installation = createInstallation({ plugins: ref(PLUGINS) })

const meta: Meta<typeof PluginsList> = {
  title: 'Window/Plugin list',
  component: PluginsList,
  parameters: { layout: 'fullscreen' },
  render: () => ({
    components: { PluginsList, Button },
    setup: () => ({ state: { installation }, plugins: PLUGINS }),
    template: `
      <div class="numen" :style="{ inlineSize: '40rem', padding: '1rem', background: 'var(--numen-surface)' }">
        <PluginsList :state="state" :plugins="plugins" />
        <Button>Install</Button>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<typeof meta>

/** The horizontal radius of the top-left corner, in pixels. */
const readCornerPx = (element: Element): number =>
  Number.parseFloat(getComputedStyle(element).borderTopLeftRadius)

/** The search field and a button have rounded corners under the default theme. */
export const TheCornersAreRounded: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const search = canvas.getByRole('searchbox', { name: 'Search plugins' })
    const button = canvas.getByRole('button', { name: 'Install' })

    await expect(readCornerPx(search)).toBeGreaterThan(0)
    await expect(readCornerPx(button)).toBeGreaterThan(0)
  },
}

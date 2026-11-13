/**
 * The title bar item that counts the running plugins, with its list open. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { PluginProcess } from '../types'
import PluginsIndicator from './PluginsIndicator.vue'
import '../styles.css'

const NOW = Date.UTC(2026, 10, 13, 9, 30, 0)
const MINUTE = 60_000

/** Made-up plugins: one in use, one idle that stops soon, one idle whose stop is due, one in the core. */
const PLUGINS: readonly PluginProcess[] = [
  {
    pluginId: 'org.example.atlas',
    pluginName: 'Atlas',
    startTime: NOW - 42 * MINUTE,
    lastUseTime: NOW - 2_000,
    leaseCount: 2,
    isInProcess: false,
    health: 'warning',
    healthText: 'The tiles are old',
  },
  {
    pluginId: 'org.example.gazette',
    pluginName: 'Gazette',
    startTime: NOW - 5 * MINUTE,
    lastUseTime: NOW - 65_000,
    leaseCount: 0,
    isInProcess: false,
    idleStopTime: NOW + 115_000,
    health: 'ok',
    healthText: '',
  },
  {
    pluginId: 'org.example.maps',
    pluginName: 'Maps',
    startTime: NOW - 3 * 60 * MINUTE,
    lastUseTime: NOW - 3 * MINUTE - 10_000,
    leaseCount: 0,
    isInProcess: false,
    idleStopTime: NOW - 10_000,
    health: 'error',
    healthText: 'The map server does not answer',
  },
  {
    pluginId: 'org.example.words',
    pluginName: 'Words',
    startTime: NOW - 3 * 60 * MINUTE,
    lastUseTime: NOW - 3 * 60 * MINUTE,
    leaseCount: 1,
    isInProcess: true,
    health: 'ok',
    healthText: '',
  },
]

const meta = {
  title: 'Running plugins/Indicator',
  component: PluginsIndicator,
  parameters: { layout: 'fullscreen' },
  args: { processes: PLUGINS, phase: 'ready', now: NOW, locale: 'en-US' },
  render: (args) => ({
    components: { PluginsIndicator },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:flex-start;justify-content:flex-end;block-size:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <header style="display:flex;align-items:center;justify-content:flex-end;position:relative;inline-size:100%;block-size:2.25rem;background:var(--numen-surface);border-bottom:1px solid var(--numen-rule);font-size:var(--numen-text-1)">
          <PluginsIndicator v-bind="args" />
        </header>
      </div>
    `,
  }),
} satisfies Meta<typeof PluginsIndicator>

export default meta
type Story = StoryObj<typeof meta>

/** The item counts four plugins; the list names each with its health and its times. */
export const Open: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: '4 plugins running' }))
    const popover = await canvas.findByRole('dialog', { name: 'Running plugins' })

    const gazette = within(within(popover).getByRole('listitem', { name: 'Gazette' }))
    await expect(gazette.getByText('Idle for')).toBeVisible()
    await expect(gazette.getByText('1m 05s')).toBeVisible()
    await expect(gazette.getByText('1m 55s')).toBeVisible()

    const atlas = within(within(popover).getByRole('listitem', { name: 'Atlas' }))
    await expect(atlas.getByText('2 calls, streams or tasks')).toBeVisible()
    await expect(atlas.getByText('The tiles are old')).toBeVisible()
    await expect(atlas.queryByText('Stops in')).toBeNull()

    const maps = within(within(popover).getByRole('listitem', { name: 'Maps' }))
    await expect(maps.getByText('Stops at the next check')).toBeVisible()

    const words = within(within(popover).getByRole('listitem', { name: 'Words' }))
    await expect(words.getByText('Runs inside the core')).toBeVisible()
  },
}

/** With no plugin running the item shows zero and the list says so. */
export const NoneRunning: Story = {
  args: { processes: [] },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: 'No plugin is running' }))
    await expect(await canvas.findByText('No plugin is running', { selector: 'p' })).toBeVisible()
  },
}

/** Without the list of the core the item shows a dash and opens nothing. */
export const Unavailable: Story = {
  args: { processes: [], phase: 'unavailable' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = await canvas.findByRole('button', {
      name: 'The running plugins are not available',
    })
    await expect(trigger.textContent?.trim()).toBe('◈–')
    await userEvent.click(trigger)
    await expect(canvas.queryByRole('dialog')).toBeNull()
  },
}

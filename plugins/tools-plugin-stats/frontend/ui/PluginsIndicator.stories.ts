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

const HEALTHY = { health: 'ok', healthText: '' } as const

const STARTED_BY_SETTINGS = { requester: 'Settings', reason: 'list tools' } as const

/**
 * Made-up plugins, one per state: in use, idle, about to stop, inside the core, and unhealthy.
 */
const PLUGINS: readonly PluginProcess[] = [
  {
    pluginId: 'org.example.atlas',
    pluginName: 'Atlas',
    startTime: NOW - 42 * MINUTE,
    lastUseTime: NOW - 2_000,
    leaseCount: 2,
    isInProcess: false,
    startedBy: { requester: 'Search', reason: 'search by meaning' },
    holders: [
      { requester: 'Search', reason: 'search by meaning' },
      { requester: 'Orchard keeper', reason: 'call tool find_place' },
    ],
    ...HEALTHY,
  },
  {
    pluginId: 'org.example.gazette',
    pluginName: 'Gazette',
    startTime: NOW - 5 * MINUTE,
    lastUseTime: NOW - 57_000,
    leaseCount: 0,
    isInProcess: false,
    idleStopTime: NOW + 122_000,
    startedBy: STARTED_BY_SETTINGS,
    holders: [],
    ...HEALTHY,
  },
  {
    pluginId: 'org.example.ledger',
    pluginName: 'Ledger',
    startTime: NOW - 3 * 60 * MINUTE,
    lastUseTime: NOW - 3 * MINUTE - 10_000,
    leaseCount: 0,
    isInProcess: false,
    idleStopTime: NOW - 10_000,
    startedBy: { requester: 'Events', reason: 'deliver onStartup' },
    holders: [],
    ...HEALTHY,
  },
  {
    pluginId: 'org.example.maps',
    pluginName: 'Maps',
    startTime: NOW - 20 * MINUTE,
    lastUseTime: NOW - 4_000,
    leaseCount: 1,
    isInProcess: false,
    startedBy: { requester: 'Commands', reason: 'run command maps.open' },
    holders: [{ requester: 'Maps', reason: 'task: Draw the tiles' }],
    health: 'error',
    healthText: 'The map server does not answer since the last restart of the tile service',
  },
  {
    pluginId: 'org.example.words',
    pluginName: 'Words',
    startTime: NOW - 3 * 60 * MINUTE,
    lastUseTime: NOW - 3 * 60 * MINUTE,
    leaseCount: 1,
    isInProcess: true,
    startedBy: STARTED_BY_SETTINGS,
    holders: [STARTED_BY_SETTINGS],
    ...HEALTHY,
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

const getRowOf = (popover: HTMLElement, name: string) =>
  within(popover).getByRole('listitem', { name })

/**
 * The item counts five plugins; the list gives each one line: a dot for in use or idle, the name,
 * the worst health item of an unhealthy plugin, and the time until an idle one stops.
 */
export const Open: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'plugins', width: 640, height: 520, drawn: 3_000 },
    ],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const page = within(canvasElement.ownerDocument.body)
    await userEvent.click(await canvas.findByRole('button', { name: '5 plugins running' }))
    const popover = await canvas.findByRole('dialog', { name: 'Running plugins' })

    const atlas = within(getRowOf(popover, 'Atlas'))
    await expect(atlas.getByRole('img', { name: 'In use' })).toBeVisible()
    await expect(atlas.queryByText(/\d+s$/)).toBeNull()

    const gazette = within(getRowOf(popover, 'Gazette'))
    await expect(gazette.getByRole('img', { name: 'Idle' })).toBeVisible()
    const countdown = gazette.getByText('2m 02s')
    await expect(countdown).toBeVisible()
    await userEvent.hover(countdown)
    await expect(page.getByRole('tooltip')).toHaveTextContent('Stops in 2m 02s; idle for 57s')

    const ledger = getRowOf(popover, 'Ledger')
    await expect(within(ledger).getByRole('img', { name: 'Idle' })).toBeVisible()
    await expect(within(ledger).getByText('stopping')).toBeVisible()
    await userEvent.hover(ledger)
    await expect(page.getByRole('tooltip')).toHaveTextContent(
      'org.example.ledger Running for 3h 00m',
    )

    const maps = within(getRowOf(popover, 'Maps'))
    const health = maps.getByText(
      'The map server does not answer since the last restart of the tile service',
    )
    await expect(health).toBeVisible()
    await expect(health).not.toHaveAttribute('title')

    const words = getRowOf(popover, 'Words')
    await expect(within(words).getByText('in core')).toBeVisible()
    await userEvent.hover(words)
    await expect(page.getByRole('tooltip')).toHaveTextContent(
      'org.example.words Running for 3h 00m Runs inside the core',
    )
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

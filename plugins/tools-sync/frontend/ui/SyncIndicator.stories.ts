import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { VaultsClient } from '../api/vaults'
import { createSyncState } from '../model/sync'
import { WORDS } from '../words'
import SyncIndicator from './SyncIndicator.vue'
import { NO_TRANSFER } from './testing/vaultFixture'
import '../styles.css'

const createClient = (
  vaults: Awaited<ReturnType<VaultsClient['listVaults']>>['vaults'],
): Pick<VaultsClient, 'getStatus' | 'listVaults'> => ({
  getStatus: () => Promise.resolve({ hasIdentity: true, isReachable: true, items: [] }),
  listVaults: () => Promise.resolve({ vaults }),
})

const work = {
  id: 'work',
  name: 'Work',
  path: '/home/person/Work',
  state: 'connected',
  lastSyncedAt: new Date(Date.now() - 5 * 60_000).toISOString(),
  conflictCount: 0,
  remoteId: 'aa11',
  ...NO_TRANSFER,
} as const

const state = createSyncState(
  createClient([
    work,
    { ...work, id: 'home', name: 'Home', syncing: true, syncedBytes: 1, totalBytes: 4 },
  ]),
)

const meta = {
  title: 'NumenSync/Indicator',
  component: SyncIndicator,
  parameters: { layout: 'fullscreen' },
  args: { state },
  render: () => ({
    components: { SyncIndicator },
    setup: () => {
      state.start()
      return { state }
    },
    template: `
      <div class="numen" style="display:flex;align-items:flex-start;justify-content:flex-end;block-size:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <header style="display:flex;align-items:center;justify-content:flex-end;position:relative;inline-size:100%;block-size:2.25rem;background:var(--numen-surface);border-bottom:1px solid var(--numen-rule);font-size:var(--numen-text-1)">
          <SyncIndicator :state="state" />
        </header>
      </div>
    `,
  }),
} satisfies Meta<typeof SyncIndicator>

export default meta
type Story = StoryObj<typeof meta>

/** The popover has a header with the title and a close button, and the vault rows under it. */
export const Open: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: /Sync/ }))
    const popover = await canvas.findByRole('dialog', { name: WORDS.vaultsTitle })
    await expect(within(popover).getByText(WORDS.vaultsTitle)).toBeVisible()
    await expect(within(popover).getByText('Work')).toBeVisible()
    await expect(
      within(popover).getByRole('progressbar', { name: WORDS.syncProgressName }),
    ).toBeVisible()
    await userEvent.click(within(popover).getByRole('button', { name: WORDS.close }))
    await expect(canvas.queryByRole('dialog')).toBeNull()
  },
}

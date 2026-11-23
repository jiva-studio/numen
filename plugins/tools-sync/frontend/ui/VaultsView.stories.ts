import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { RemoteVault, Vault, VaultsClient } from '../api/vaults'
import VaultsView from './VaultsView.vue'
import '../styles.css'

const NOW = new Date('2026-11-23T12:00:00Z')

const connectedVault: Vault = {
  id: 'work',
  name: 'Work',
  path: '/home/person/Work',
  state: 'connected',
  lastSyncedAt: '2026-11-23T11:55:00Z',
  conflictCount: 0,
}

const plainVault: Vault = {
  id: 'plain',
  name: 'Plain',
  path: '/home/person/Plain',
  state: 'notConnected',
  lastSyncedAt: '',
  conflictCount: 0,
}

const remoteVaults: RemoteVault[] = [
  { remoteId: 'aa11', name: 'Work', isConnected: true },
  { remoteId: 'bb22', name: 'Home', isConnected: false },
]

interface Setup {
  readonly vaults: Vault[]
  readonly remote?: RemoteVault[]
  readonly hasIdentity?: boolean
  readonly refusal?: string
}

const createClient = (setup: Setup): VaultsClient => ({
  listVaults: () => Promise.resolve({ vaults: setup.vaults }),
  listRemoteVaults: () => Promise.resolve({ vaults: setup.remote ?? [] }),
  resolveDeletions: () => Promise.resolve(),
  connectVault: () =>
    setup.refusal === undefined ? Promise.resolve() : Promise.reject(new Error(setup.refusal)),
  getStatus: () =>
    Promise.resolve({ hasIdentity: setup.hasIdentity ?? true, isReachable: true, items: [] }),
})

const meta = {
  title: 'NumenSync/Vaults',
  component: VaultsView,
  render: (args) => ({
    components: { VaultsView },
    setup: () => ({ args }),
    template: '<VaultsView v-bind="args" />',
  }),
} satisfies Meta<typeof VaultsView>

export default meta
type Story = StoryObj<typeof meta>

export const NoIdentity: Story = {
  args: { client: createClient({ vaults: [plainVault], hasIdentity: false }), now: () => NOW },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('No identity on this device')).toBeVisible()
    await expect(canvas.queryByRole('button', { name: 'Connect as a new synced vault' })).toBeNull()
  },
}

export const EmptyList: Story = {
  args: { client: createClient({ vaults: [] }), now: () => NOW },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Vaults')).toBeVisible()
    await expect(canvas.queryByRole('button')).toBeNull()
  },
}

export const ConnectedAndNotConnected: Story = {
  args: { client: createClient({ vaults: [connectedVault, plainVault] }), now: () => NOW },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Synced 5 minutes ago')).toBeVisible()
    await expect(canvas.getByText('Not connected')).toBeVisible()
    await expect(
      canvas.getByRole('button', { name: 'Connect as a new synced vault' }),
    ).toBeVisible()
  },
}

export const RemoteListOpen: Story = {
  args: { client: createClient({ vaults: [plainVault], remote: remoteVaults }), now: () => NOW },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(
      await canvas.findByRole('button', { name: 'Connect to an existing synced vault' }),
    )
    await expect(await canvas.findByText('Choose a synced vault')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Home' })).toBeVisible()
  },
}

export const CopyRefused: Story = {
  args: {
    client: createClient({ vaults: [plainVault], refusal: 'a copy of a connected vault' }),
    now: () => NOW,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(
      await canvas.findByRole('button', { name: 'Connect as a new synced vault' }),
    )
    await expect(await canvas.findByRole('alert')).toHaveTextContent('a copy of a connected vault')
  },
}

export const DeletionsPaused: Story = {
  args: {
    client: createClient({
      vaults: [
        { ...connectedVault, pausedDeletions: { count: 5, tracked: 10, direction: 'send' } },
      ],
    }),
    now: () => NOW,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Sync is paused: 5 of 10 files are removed')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Remove the files' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Keep the files' })).toBeVisible()
  },
}

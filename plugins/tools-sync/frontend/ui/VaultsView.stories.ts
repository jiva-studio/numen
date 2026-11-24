import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import type { RemoteVault, Vault, VaultsClient } from '../api/vaults'
import { frameDecorator } from './storyFrame'
import VaultsView from './VaultsView.vue'
import '../styles.css'

const NOW = new Date('2026-11-23T12:00:00Z')

const work: Vault = {
  id: 'work',
  name: 'Work',
  path: '/home/person/Work',
  state: 'connected',
  lastSyncedAt: '2026-11-23T11:55:00Z',
  conflictCount: 0,
}

const home: Vault = { ...work, id: 'home', name: 'Home', lastSyncedAt: '2026-11-23T11:59:50Z' }

const plain: Vault = {
  id: 'plain',
  name: 'Plain',
  path: '/home/person/Plain',
  state: 'notConnected',
  lastSyncedAt: '',
  conflictCount: 0,
}

const cloud: RemoteVault[] = [
  { remoteId: 'aa11', name: 'Work', isConnected: true },
  { remoteId: 'bb22', name: 'Recipes', isConnected: false },
]

interface Setup {
  readonly vaults?: Vault[]
  readonly remote?: RemoteVault[]
  readonly hasIdentity?: boolean
  readonly isReachable?: boolean
  readonly differCount?: number
  readonly failure?: string
  readonly isPending?: boolean
}

const never = <T>(): Promise<T> => new Promise<T>(() => undefined)

const createClient = (setup: Setup): VaultsClient => ({
  listVaults: () => Promise.resolve({ vaults: setup.vaults ?? [] }),
  listRemoteVaults: () => Promise.resolve({ vaults: setup.remote ?? [] }),
  getConnectPreview: () => Promise.resolve({ differCount: setup.differCount ?? 0 }),
  resolveDeletions: () => Promise.resolve(),
  connectVault: () => Promise.resolve(),
  stopSyncingVault: () => Promise.resolve(),
  getStatus: () => {
    if (setup.isPending === true) return never()
    if (setup.failure !== undefined) return Promise.reject(new Error(setup.failure))
    return Promise.resolve({
      hasIdentity: setup.hasIdentity ?? true,
      isReachable: setup.isReachable ?? true,
      items: [],
    })
  },
})

const meta = {
  title: 'NumenSync/Vaults',
  component: VaultsView,
  decorators: [frameDecorator],
  render: (args) => ({
    components: { VaultsView },
    setup: () => ({ args }),
    template: '<VaultsView v-bind="args" />',
  }),
} satisfies Meta<typeof VaultsView>

export default meta
type Story = StoryObj<typeof meta>

export const Loading: Story = {
  args: { client: createClient({ isPending: true }), now: () => NOW },
}

export const LoadFailed: Story = {
  args: { client: createClient({ failure: 'The sync service did not answer' }), now: () => NOW },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent('did not answer')
    await expect(canvas.getByRole('button', { name: 'Try again' })).toBeVisible()
  },
}

export const Empty: Story = {
  args: { client: createClient({ vaults: [plain] }), now: () => NOW },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('No vault is synced yet')).toBeVisible()
    await expect(canvas.queryByText('Plain')).toBeNull()
    await expect(canvas.queryByRole('button')).toBeNull()
  },
}

export const Synced: Story = {
  args: { client: createClient({ vaults: [work, home, plain] }), now: () => NOW },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Work')).toBeVisible()
    await expect(canvas.getAllByText('Up to date')).toHaveLength(2)
    await expect(canvas.getByText('Last sync 5 minutes ago')).toBeVisible()
    await expect(canvas.queryByText('Plain')).toBeNull()
  },
}

export const ChangedOnTwoDevices: Story = {
  args: { client: createClient({ vaults: [{ ...work, conflictCount: 3 }] }), now: () => NOW },
  play: async ({ canvasElement }) => {
    await expect(
      await within(canvasElement).findByText('3 notes changed on two devices'),
    ).toBeVisible()
  },
}

export const DeletionsPaused: Story = {
  args: {
    client: createClient({
      vaults: [{ ...work, pausedDeletions: { count: 5, tracked: 10, direction: 'send' } }],
    }),
    now: () => NOW,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Paused: 5 of 10 files are removed here')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Remove the files' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Keep the files' })).toBeVisible()
  },
}

export const ServerAway: Story = {
  args: { client: createClient({ vaults: [work], isReachable: false }), now: () => NOW },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText('Waiting for the server')).toBeVisible()
  },
}

export const NoIdentity: Story = {
  args: { client: createClient({ hasIdentity: false }), now: () => NOW },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText('No identity on this device')).toBeVisible()
  },
}

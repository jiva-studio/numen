import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { RemoteVault, Vault, VaultsClient } from '../api/vaults'
import GetVaultDialog from './GetVaultDialog.vue'
import { frameDecorator } from './storyFrame'
import '../styles.css'

const vault = (id: string, state: Vault['state']): Vault => ({
  id,
  name: id,
  path: `/home/person/${id}`,
  state,
  lastSyncedAt: '',
  conflictCount: 0,
})

const cloud: RemoteVault[] = [
  { remoteId: 'aa11', name: 'Work', isConnected: true },
  { remoteId: 'bb22', name: 'Recipes', isConnected: false },
]

const createClient = (vaults: Vault[], differCount = 0): VaultsClient => ({
  listVaults: () => Promise.resolve({ vaults }),
  listRemoteVaults: () => Promise.resolve({ vaults: cloud }),
  getConnectPreview: () => Promise.resolve({ differCount }),
  resolveDeletions: () => Promise.resolve(),
  connectVault: () => Promise.resolve(),
  stopSyncingVault: () => Promise.resolve(),
  getStatus: () => Promise.resolve({ hasIdentity: true, isReachable: true, items: [] }),
})

const meta = {
  title: 'NumenSync/GetVaultDialog',
  component: GetVaultDialog,
  decorators: [frameDecorator],
  render: (args) => ({
    components: { GetVaultDialog },
    setup: () => ({ args }),
    template: '<GetVaultDialog v-bind="args" />',
  }),
} satisfies Meta<typeof GetVaultDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Choose: Story = {
  args: { client: createClient([vault('Plain', 'notConnected')]) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByLabelText('Vault in the cloud')).toBeVisible()
    await expect(canvas.getByLabelText('Vault on this device')).toBeVisible()
  },
}

export const NoVaultOnDevice: Story = {
  args: { client: createClient([vault('Work', 'connected')]) },
  play: async ({ canvasElement }) => {
    await expect(
      await within(canvasElement).findByText(
        'Every vault on this device is synced. Create an empty vault first.',
      ),
    ).toBeVisible()
  },
}

export const Asks: Story = {
  args: { client: createClient([vault('Plain', 'notConnected')], 2) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: 'Get vault' }))
    await expect(
      await canvas.findByText('2 notes differ from the vault in the cloud. Get it anyway?'),
    ).toBeVisible()
  },
}

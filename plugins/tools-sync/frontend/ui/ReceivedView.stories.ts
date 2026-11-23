import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { ReceivedShare, SharesClient } from '../api/shares'
import ReceivedView from './ReceivedView.vue'
import '../styles.css'

const OWNER = 'EEEE FFFF GGGG'

const food = (state: ReceivedShare['state']): ReceivedShare => ({
  vaultId: 'r1',
  name: 'Food',
  ownerFingerprint: OWNER,
  state,
  lastSyncedAt: '',
})

interface Setup {
  readonly received?: ReceivedShare[]
  readonly invitationRefusal?: string
  readonly acceptRefusal?: string
}

const createClient = (setup: Setup): SharesClient => ({
  getReaderCode: () => Promise.resolve({ code: 'numen-reader-1.abc', fingerprint: 'AAAA BBBB' }),
  readReaderCode: () => Promise.reject(new Error('unused')),
  getShare: () => Promise.reject(new Error('unused')),
  addReader: () => Promise.reject(new Error('unused')),
  removeReader: () => Promise.reject(new Error('unused')),
  stopShare: () => Promise.reject(new Error('unused')),
  readInvitation: () =>
    setup.invitationRefusal === undefined
      ? Promise.resolve({ shareName: 'Food', ownerFingerprint: OWNER, isForThisAccount: true, alreadyReceived: false })
      : Promise.reject(new Error(setup.invitationRefusal)),
  acceptInvitation: () =>
    setup.acceptRefusal === undefined ? Promise.resolve() : Promise.reject(new Error(setup.acceptRefusal)),
  listReceived: () => Promise.resolve(setup.received ?? []),
})

const meta = {
  title: 'NumenSync/Shares',
  component: ReceivedView,
  render: (args) => ({
    components: { ReceivedView },
    setup: () => ({ args }),
    template: '<ReceivedView v-bind="args" />',
  }),
} satisfies Meta<typeof ReceivedView>

export default meta
type Story = StoryObj<typeof meta>

export const NothingReceived: Story = {
  args: { client: createClient({}) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('textbox', { name: 'Your code' })).toHaveValue('numen-reader-1.abc')
    await expect(canvas.getByText('No shared folders received yet')).toBeVisible()
  },
}

export const Ready: Story = {
  args: { client: createClient({ received: [food('ready')] }) },
  play: async ({ canvasElement }) => {
    const list = await within(canvasElement).findByRole('list', { name: 'Received shares' })
    await expect(within(list).getByText('Up to date')).toBeVisible()
  },
}

export const Ended: Story = {
  args: { client: createClient({ received: [food('ended')] }) },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText('The owner ended this share')).toBeVisible()
  },
}

export const InvitationPasted: Story = {
  args: { client: createClient({}) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(await canvas.findByRole('textbox', { name: 'Invitation' }), 'numen-invitation-1.xyz')
    await expect(await canvas.findByText(OWNER)).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Receive' })).toBeEnabled()
  },
}

export const InvitationOfAnotherPerson: Story = {
  args: { client: createClient({ invitationRefusal: 'This invitation is for another person' }) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(await canvas.findByRole('textbox', { name: 'Invitation' }), 'numen-invitation-1.xyz')
    await expect(await canvas.findByRole('alert')).toHaveTextContent('This invitation is for another person')
  },
}

export const OwnerIdentityChanged: Story = {
  args: { client: createClient({ acceptRefusal: 'The owner identity changed' }) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(await canvas.findByRole('textbox', { name: 'Invitation' }), 'numen-invitation-1.xyz')
    await userEvent.click(canvas.getByRole('button', { name: 'Receive' }))
    await expect(await canvas.findByRole('alert')).toHaveTextContent('The owner identity changed')
  },
}

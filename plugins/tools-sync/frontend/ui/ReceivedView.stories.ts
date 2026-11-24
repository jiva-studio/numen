import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { FolderSharesClient, ReceivedShare, ReceivedSharesClient } from '../api/shares'
import { createReceivedPage } from './receivedView'
import { frameDecorator } from './storyFrame'
import '../styles.css'

const OWNER = 'EEEE FFFF GGGG HHHH IIII'

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

type SharesClient = FolderSharesClient & ReceivedSharesClient

const createClient = (setup: Setup): SharesClient => ({
  getReaderCode: () => Promise.resolve({ code: 'numen-reader-1.abc', fingerprint: 'AAAA BBBB' }),
  readReaderCode: () => Promise.reject(new Error('unused')),
  getShare: () => Promise.reject(new Error('unused')),
  addReader: () => Promise.reject(new Error('unused')),
  removeReader: () => Promise.reject(new Error('unused')),
  stopShare: () => Promise.reject(new Error('unused')),
  readInvitation: () =>
    setup.invitationRefusal === undefined
      ? Promise.resolve({
          shareName: 'Food',
          ownerFingerprint: OWNER,
          isForThisAccount: true,
          alreadyReceived: false,
        })
      : Promise.reject(new Error(setup.invitationRefusal)),
  acceptInvitation: () =>
    setup.acceptRefusal === undefined
      ? Promise.resolve()
      : Promise.reject(new Error(setup.acceptRefusal)),
  listReceived: () => Promise.resolve(setup.received ?? []),
})

const createPage = (setup: Setup) => createReceivedPage({ client: createClient(setup) })

const meta = {
  title: 'NumenSync/SharedFolders',
  decorators: [frameDecorator],
  render: (args) => ({
    components: { Page: args.page },
    template: '<Page />',
  }),
} satisfies Meta<{ page: ReturnType<typeof createReceivedPage> }>

export default meta
type Story = StoryObj<typeof meta>

const openInvitation = async (canvasElement: HTMLElement): Promise<ReturnType<typeof within>> => {
  const canvas = within(canvasElement)
  await userEvent.click(await canvas.findByRole('button', { name: 'Accept an invitation' }))
  await userEvent.type(await canvas.findByRole('textbox', { name: 'Invitation' }), 'numen-invitation-1.xyz')
  return canvas
}

export const Empty: Story = {
  args: { page: createPage({}) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('No folder is shared yet')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Copy my sharing link' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Accept an invitation' })).toBeVisible()
  },
}

export const Ready: Story = {
  args: { page: createPage({ received: [food('ready')] }) },
  play: async ({ canvasElement }) => {
    const list = await within(canvasElement).findByRole('list', { name: 'Shared with you' })
    await expect(within(list).getByText('Up to date')).toBeVisible()
    await expect(within(list).getByText('Read only · From EEEE FFFF GGGG…')).toBeVisible()
  },
}

export const Receiving: Story = {
  args: { page: createPage({ received: [food('receiving')] }) },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText('Getting the folder…')).toBeVisible()
  },
}

export const Ended: Story = {
  args: { page: createPage({ received: [food('ended')] }) },
  play: async ({ canvasElement }) => {
    await expect(
      await within(canvasElement).findByText('The person stopped sharing this folder'),
    ).toBeVisible()
  },
}

export const AcceptDialog: Story = {
  args: { page: createPage({}) },
  play: async ({ canvasElement }) => {
    const canvas = await openInvitation(canvasElement)
    await expect(await canvas.findByText(/Food: From/)).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Accept' })).toBeEnabled()
  },
}

export const InvitationOfAnotherPerson: Story = {
  args: { page: createPage({ invitationRefusal: 'This invitation is for another person' }) },
  play: async ({ canvasElement }) => {
    const canvas = await openInvitation(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent(
      'This invitation is for another person',
    )
  },
}

export const OwnerIdentityChanged: Story = {
  args: { page: createPage({ acceptRefusal: 'The owner identity changed' }) },
  play: async ({ canvasElement }) => {
    const canvas = await openInvitation(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Accept' }))
    await expect(await canvas.findByRole('alert')).toHaveTextContent('The owner identity changed')
  },
}

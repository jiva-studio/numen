import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { SharesClient, Share, ShareReader } from '../api/shares'
import { createShareView } from '../model/shares'
import ShareDialog from './ShareDialog.vue'
import '../styles.css'

const TARGET = { vaultId: 'v1', folder: 'Food' }
const CODE = 'numen-reader-1.abc'
const FINGERPRINT = 'CCCC DDDD EEEE'
const INVITATION = 'numen-invitation-1.xyz'

const anna: ShareReader = { label: 'Anna', fingerprint: FINGERPRINT, state: 'active' }

interface Setup {
  readonly share?: Share
  readonly refusal?: string
  readonly label?: string
}

const createClient = (setup: Setup): SharesClient => ({
  getReaderCode: () => Promise.resolve({ code: CODE, fingerprint: 'AAAA' }),
  readReaderCode: () =>
    Promise.resolve({ fingerprint: FINGERPRINT, ...(setup.label === undefined ? {} : { label: setup.label }) }),
  getShare: () => Promise.resolve(setup.share ?? { state: 'notShared', readers: [] }),
  addReader: () =>
    setup.refusal === undefined
      ? Promise.resolve({ invitation: INVITATION, fingerprint: FINGERPRINT })
      : Promise.reject(new Error(setup.refusal)),
  removeReader: () => Promise.resolve(),
  stopShare: () => Promise.resolve(),
  readInvitation: () => Promise.reject(new Error('unused')),
  acceptInvitation: () => Promise.resolve(),
  listReceived: () => Promise.resolve([]),
})

const createState = (setup: Setup) => {
  const state = createShareView({ client: createClient(setup), target: TARGET, close: () => undefined })
  state.load().catch(state.fail)
  return state
}

const meta = {
  title: 'NumenSync/ShareDialog',
  component: ShareDialog,
  render: (args) => ({
    components: { ShareDialog },
    setup: () => ({ args }),
    template: '<ShareDialog v-bind="args" />',
  }),
} satisfies Meta<typeof ShareDialog>

export default meta
type Story = StoryObj<typeof meta>

const fillAndShare = async (canvasElement: HTMLElement): Promise<ReturnType<typeof within>> => {
  const canvas = within(canvasElement)
  await userEvent.type(await canvas.findByRole('textbox', { name: 'Reader code' }), CODE)
  await expect(await canvas.findByLabelText('Reader account fingerprint')).toHaveTextContent(FINGERPRINT)
  await userEvent.type(canvas.getByRole('textbox', { name: 'Label' }), 'Anna')
  await userEvent.click(canvas.getByRole('button', { name: 'Share' }))
  return canvas
}

export const NotShared: Story = {
  args: { state: createState({}) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('textbox', { name: 'Reader code' })).toBeVisible()
    await expect(canvas.queryByRole('list', { name: 'Readers' })).toBeNull()
    await expect(canvas.queryByRole('button', { name: 'Stop sharing' })).toBeNull()
  },
}

export const Creating: Story = {
  args: { state: createState({ share: { state: 'creating', readers: [] } }) },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText('Preparing the shared folder…')).toBeVisible()
  },
}

export const InvitationShown: Story = {
  args: { state: createState({}) },
  play: async ({ canvasElement }) => {
    const canvas = await fillAndShare(canvasElement)
    await expect(await canvas.findByRole('textbox', { name: 'Invitation' })).toHaveValue(INVITATION)
    await expect(canvas.getByText(/only the reader can use it/)).toBeVisible()
  },
}

export const ActiveWithReaders: Story = {
  args: { state: createState({ share: { state: 'active', readers: [anna] } }) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const readers = await canvas.findByRole('list', { name: 'Readers' })
    await expect(within(readers).getByText('Anna')).toBeVisible()
    await expect(within(readers).getByRole('button', { name: 'Remove' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Stop sharing' })).toBeVisible()
    await expect(canvas.getByText('A removed reader keeps what the reader already has.')).toBeVisible()
  },
}

export const FolderInsideSharedFolder: Story = {
  args: { state: createState({ refusal: 'A folder inside a shared folder cannot be shared' }) },
  play: async ({ canvasElement }) => {
    const canvas = await fillAndShare(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent('A folder inside a shared folder cannot be shared')
  },
}

export const ReaderNotSignedIn: Story = {
  args: { state: createState({ refusal: 'The reader could not be added; the reader may not have signed in to Numen Sync yet' }) },
  play: async ({ canvasElement }) => {
    const canvas = await fillAndShare(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent('could not be added')
  },
}

export const AnotherIdentity: Story = {
  args: { state: createState({ refusal: 'This person has another identity than before' }) },
  play: async ({ canvasElement }) => {
    const canvas = await fillAndShare(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent('another identity')
    await expect(canvas.getByRole('button', { name: 'Share anyway' })).toBeVisible()
  },
}

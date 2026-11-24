import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { FolderSharesClient, ReceivedSharesClient, Share, ShareReader } from '../api/shares'
import { createShareView } from '../model/shares'
import ShareDialog from './ShareDialog.vue'
import { frameDecorator } from './storyFrame'
import '../styles.css'

const TARGET = { vaultId: 'v1', folder: 'Food' }
const CODE = 'numen-reader-1.abc'
const FINGERPRINT = 'CCCC DDDD EEEE'
const LINK = `numen://share/${CODE}`
const INVITATION = 'numen-invitation-1.xyz'

const phone: ShareReader = { label: 'Phone', fingerprint: FINGERPRINT, state: 'active' }

interface Setup {
  readonly share?: Share
  readonly refusal?: string
  readonly label?: string
  readonly isInvalid?: boolean
  readonly isPending?: boolean
}

type SharesClient = FolderSharesClient & ReceivedSharesClient

const createClient = (setup: Setup): SharesClient => ({
  getReaderCode: () => Promise.resolve({ code: CODE, fingerprint: 'AAAA' }),
  readReaderCode: () =>
    setup.isInvalid === true
      ? Promise.reject(new Error('this is not a reader code or an invitation'))
      : Promise.resolve({
      fingerprint: FINGERPRINT,
      label: setup.label ?? 'Phone',
    }),
  getShare: () => Promise.resolve(setup.share ?? { state: 'notShared', readers: [] }),
  addReader: () =>
    setup.isPending === true
      ? new Promise(() => undefined)
      : setup.refusal === undefined
      ? Promise.resolve({ invitation: INVITATION, fingerprint: FINGERPRINT })
      : Promise.reject(new Error(setup.refusal)),
  removeReader: () => Promise.resolve(),
  stopShare: () => Promise.resolve(),
  readInvitation: () => Promise.reject(new Error('unused')),
  acceptInvitation: () => Promise.resolve(),
  listReceived: () => Promise.resolve([]),
})

const createState = (setup: Setup) => {
  const state = createShareView({
    client: createClient(setup),
    target: TARGET,
    close: () => undefined,
  })
  state.load().catch(state.fail)
  return state
}

const meta = {
  title: 'NumenSync/ShareDialog',
  component: ShareDialog,
  decorators: [frameDecorator],
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
  await userEvent.type(await canvas.findByLabelText('Sharing link of the person'), LINK)
  await userEvent.click(canvas.getByRole('button', { name: 'Share the folder' }))
  return canvas
}

export const Empty: Story = {
  args: { state: createState({}) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Share Food (read only)')).toBeVisible()
    await expect(canvas.getByText('Nobody has access yet.')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Share the folder' })).toBeDisabled()
  },
}

export const InvalidCode: Story = {
  args: { state: createState({ isInvalid: true }) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(await canvas.findByLabelText('Sharing link of the person'), 'nonsense')
    await expect(await canvas.findByRole('alert')).toHaveTextContent(
      'This does not look like a sharing link.',
    )
  },
}

export const Sharing: Story = {
  args: { state: createState({ isPending: true }) },
  play: async ({ canvasElement }) => {
    const canvas = await fillAndShare(canvasElement)
    await expect(await canvas.findByText('Sharing the folder…')).toBeVisible()
  },
}

export const Shared: Story = {
  args: { state: createState({}) },
  play: async ({ canvasElement }) => {
    const canvas = await fillAndShare(canvasElement)
    await expect(await canvas.findByRole('textbox', { name: 'Invitation to send' })).toHaveValue(
      INVITATION,
    )
    await expect(
      canvas.getByText('Folder shared with Device «Phone» (CCCC DDDD EEEE…). Send them this invitation:'),
    ).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Copy invitation' })).toBeVisible()
  },
}

export const WithPeople: Story = {
  args: { state: createState({ share: { state: 'active', readers: [phone] } }) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Phone')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Remove access' })).toBeVisible()
  },
}

export const Refused: Story = {
  args: { state: createState({ refusal: 'A folder inside a shared folder cannot be shared' }) },
  play: async ({ canvasElement }) => {
    const canvas = await fillAndShare(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent(
      'A folder inside a shared folder cannot be shared',
    )
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

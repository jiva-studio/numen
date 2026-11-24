import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import type { FolderSharesClient, ReceivedSharesClient } from '../api/shares'
import { createReceivedView } from '../model/shares'
import AcceptInvitationDialog from './AcceptInvitationDialog.vue'
import { frameDecorator } from './storyFrame'
import '../styles.css'

const OWNER = 'EEEE FFFF GGGG HHHH IIII'

interface Setup {
  readonly invitationRefusal?: string
  readonly acceptRefusal?: string
}

const createClient = (setup: Setup): ReceivedSharesClient & Partial<FolderSharesClient> => ({
  getReaderCode: () => Promise.resolve({ code: 'numen-reader-1.abc', fingerprint: 'AAAA BBBB' }),
  readInvitation: () =>
    setup.invitationRefusal === undefined
      ? Promise.resolve({
          shareName: 'Recipes',
          ownerFingerprint: OWNER,
          isForThisAccount: true,
          alreadyReceived: false,
        })
      : Promise.reject(new Error(setup.invitationRefusal)),
  acceptInvitation: () =>
    setup.acceptRefusal === undefined
      ? Promise.resolve()
      : Promise.reject(new Error(setup.acceptRefusal)),
  listReceived: () => Promise.resolve([]),
})

const createView = (setup: Setup) => createReceivedView({ client: createClient(setup) })

const meta = {
  title: 'NumenSync/AcceptInvitation',
  component: AcceptInvitationDialog,
  decorators: [frameDecorator],
  render: (args) => ({
    components: { AcceptInvitationDialog },
    setup: () => ({ args }),
    template: '<AcceptInvitationDialog v-bind="args" />',
  }),
} satisfies Meta<typeof AcceptInvitationDialog>

export default meta
type Story = StoryObj<typeof meta>

const paste = async (canvasElement: HTMLElement): Promise<ReturnType<typeof within>> => {
  const canvas = within(canvasElement)
  const field = await canvas.findByRole('textbox', { name: 'Invitation' })
  await userEvent.type(field, 'numen-invitation-1.xyz')
  return canvas
}

export const Empty: Story = {
  args: { view: createView({}) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Accept an invitation')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Accept' })).toBeDisabled()
  },
}

export const Pasted: Story = {
  args: { view: createView({}) },
  play: async ({ canvasElement }) => {
    const canvas = await paste(canvasElement)
    await expect(await canvas.findByText(/Recipes: From/)).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Accept' })).toBeEnabled()
  },
}

export const ForAnotherPerson: Story = {
  args: { view: createView({ invitationRefusal: 'This invitation is for another person' }) },
  play: async ({ canvasElement }) => {
    const canvas = await paste(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent(
      'This invitation is for another person',
    )
  },
}

export const OwnerChanged: Story = {
  args: { view: createView({ acceptRefusal: 'The owner identity changed' }) },
  play: async ({ canvasElement }) => {
    const canvas = await paste(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Accept' }))
    await expect(await canvas.findByRole('alert')).toHaveTextContent('The owner identity changed')
  },
}

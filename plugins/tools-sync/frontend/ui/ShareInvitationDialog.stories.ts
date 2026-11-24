import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import type { ShareInvitationClient, ShareRequest } from '../api/invitations'
import type { FolderSharesClient, SharedFolder } from '../api/shares'
import { createInvitationView } from '../model/invitation'
import ShareInvitationDialog from './ShareInvitationDialog.vue'
import { decorateWithFrame } from './testing/storyFrame'
import '../styles.css'

type Reader = SharedFolder['readers'][number]

const TARGET = { vaultId: 'v1', folder: 'Recipes' }
const PHONE: Reader = { label: 'Phone', fingerprint: '1111 2222 3333 4444', state: 'active' }
const REQUEST: ShareRequest = {
  requestId: 'r1',
  ...TARGET,
  deviceName: 'Phone',
  fingerprint: '1111 2222 3333 4444',
  code: '482913',
}

interface Setup {
  readonly request?: ShareRequest
  readonly readers?: Reader[]
  readonly isUsed?: boolean
  readonly isPending?: boolean
  readonly failure?: string
}

const createInvitations = (setup: Setup): ShareInvitationClient => ({
  startShareInvitation: () => {
    if (setup.isPending === true) return new Promise(() => undefined)
    if (setup.failure !== undefined) return Promise.reject(new Error(setup.failure))
    return Promise.resolve({ id: 'i1', link: 'numen-share:eyJ2IjoxfQ', expiresAt: '2026-11-25T12:00:00Z' })
  },
  listShareInvitations: () =>
    Promise.resolve(
      setup.isUsed === true
        ? []
        : [{ id: 'i1', folder: 'Recipes', expiresAt: '2026-11-25T12:00:00Z', isRequested: false }],
    ),
  listShareRequests: () => Promise.resolve(setup.request === undefined ? [] : [setup.request]),
  revokeShareInvitation: () => Promise.resolve(),
  approveShareRequest: () => Promise.resolve(),
  declineShareRequest: () => Promise.resolve(),
})

const createShares = (setup: Setup): FolderSharesClient => ({
  listShares: () =>
    Promise.resolve(
      setup.readers === undefined
        ? []
        : [{ ...TARGET, state: 'active' as const, readers: setup.readers }],
    ),
  removeReader: () => Promise.resolve(),
  stopShare: () => Promise.resolve(),
})

const createState = (setup: Setup) => {
  const state = createInvitationView({
    invitations: createInvitations(setup),
    shares: createShares(setup),
    target: TARGET,
    close: () => undefined,
    refreshMs: 1_000_000,
  })
  state.start().catch(state.fail)
  return state
}

const meta = {
  title: 'NumenSync/ShareInvitationDialog',
  component: ShareInvitationDialog,
  decorators: [decorateWithFrame],
  render: (args) => ({
    components: { ShareInvitationDialog },
    setup: () => ({ args }),
    template: '<ShareInvitationDialog v-bind="args" />',
  }),
} satisfies Meta<typeof ShareInvitationDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Starting: Story = {
  args: { state: createState({ isPending: true }) },
}

export const WaitingForDevice: Story = {
  args: { state: createState({}) },
  play: async ({ canvasElement }) => {
    const screen = within(canvasElement.ownerDocument.body)
    await expect(await screen.findByRole('textbox', { name: 'Invitation link' })).toHaveValue(
      'numen-share:eyJ2IjoxfQ',
    )
    await expect(screen.getByText(/^Works once\. Expires in \d+ (min|h)\.$/)).toBeVisible()
    await expect(screen.getByText('Waiting for a device to open the link')).toBeVisible()
    await expect(screen.getByRole('button', { name: 'Copy link' })).toBeVisible()
    await expect(screen.getByRole('button', { name: 'Revoke' })).toBeVisible()
  },
}

export const DeviceAsks: Story = {
  args: { state: createState({ request: REQUEST }) },
  play: async ({ canvasElement }) => {
    const screen = within(canvasElement.ownerDocument.body)
    await expect(
      await screen.findByText(
        'Device «Phone» (1111 2222 3333…) wants to join Recipes. Code 482 913',
      ),
    ).toBeVisible()
    await expect(screen.getByRole('button', { name: 'Approve' })).toBeVisible()
    await expect(screen.getByRole('button', { name: 'Decline' })).toBeVisible()
  },
}

export const DeviceReads: Story = {
  args: { state: createState({ isUsed: true, readers: [PHONE] }) },
  play: async ({ canvasElement }) => {
    const screen = within(canvasElement.ownerDocument.body)
    await expect(await screen.findByText('Device «Phone» (1111 2222 3333…)')).toBeVisible()
    await expect(screen.getByRole('button', { name: 'Remove access' })).toBeVisible()
    await expect(screen.queryByRole('textbox', { name: 'Invitation link' })).toBeNull()
  },
}

export const Failed: Story = {
  args: { state: createState({ failure: 'The server did not answer' }) },
  play: async ({ canvasElement }) => {
    await expect(
      await within(canvasElement.ownerDocument.body).findByRole('alert'),
    ).toHaveTextContent('The server did not answer')
  },
}

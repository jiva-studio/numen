import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import type {
  InvitationState,
  ShareInvitationClient,
  SharePerson,
  ShareRequest,
} from '../api/invitations'
import { createInvitationView } from '../model/invitation'
import ShareInvitationDialog from './ShareInvitationDialog.vue'
import { frameDecorator } from './storyFrame'
import '../styles.css'

const PHONE: SharePerson = { deviceName: 'Phone', fingerprint: '1111 2222 3333 4444' }
const REQUEST: ShareRequest = {
  requestId: 'r1',
  deviceName: 'Phone',
  fingerprint: '1111 2222 3333 4444',
  folderName: 'Recipes',
  code: '482 913',
}

interface Setup {
  readonly state?: InvitationState
  readonly request?: ShareRequest
  readonly people?: SharePerson[]
  readonly isPending?: boolean
  readonly failure?: string
}

const createClient = (setup: Setup): ShareInvitationClient => ({
  startShareInvitation: () => {
    if (setup.isPending === true) return new Promise(() => undefined)
    if (setup.failure !== undefined) return Promise.reject(new Error(setup.failure))
    return Promise.resolve({ link: 'numen://invite/3f9a1c', expiresAt: '2026-11-25T12:00:00Z' })
  },
  getInvitationStatus: () =>
    Promise.resolve({
      state: setup.state ?? 'waiting',
      requests: setup.request === undefined ? [] : [setup.request],
      people: setup.people ?? [],
    }),
  approveShareRequest: () => Promise.resolve(),
  declineShareRequest: () => Promise.resolve(),
  revokeShareInvitation: () => Promise.resolve(),
})

const createState = (setup: Setup) => {
  const state = createInvitationView({
    client: createClient(setup),
    target: { vaultId: 'v1', folder: 'Recipes' },
    close: () => undefined,
    refreshMs: 1_000_000,
  })
  state.start().catch(state.fail)
  return state
}

const meta = {
  title: 'NumenSync/ShareInvitationDialog',
  component: ShareInvitationDialog,
  decorators: [frameDecorator],
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

export const LinkShown: Story = {
  args: { state: createState({}) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('textbox', { name: 'Invitation link' })).toHaveValue(
      'numen://invite/3f9a1c',
    )
    await expect(canvas.getByText('Valid for 24 hours, works once.')).toBeVisible()
    await expect(canvas.getByText('Waiting for the person to open the link')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Revoke the link' })).toBeVisible()
  },
}

export const RequestWaiting: Story = {
  args: { state: createState({ request: REQUEST }) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByText('Device «Phone» (1111 2222 3333…) wants to read Recipes.'),
    ).toBeVisible()
    await expect(canvas.getByText('Code 482 913')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Allow' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Decline' })).toBeVisible()
  },
}

export const Approved: Story = {
  args: { state: createState({ state: 'approved', people: [PHONE] }) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByText('Shared with Device «Phone» (1111 2222 3333…)'),
    ).toBeVisible()
    await expect(canvas.getAllByText('1111 2222 3333…')).toHaveLength(2)
  },
}

export const Declined: Story = {
  args: { state: createState({ state: 'declined' }) },
  play: async ({ canvasElement }) => {
    await expect(
      await within(canvasElement).findByText('You declined the request. Nothing is shared.'),
    ).toBeVisible()
  },
}

export const Expired: Story = {
  args: { state: createState({ state: 'expired' }) },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText(/The link expired/)).toBeVisible()
  },
}

export const Failed: Story = {
  args: { state: createState({ failure: 'The server did not answer' }) },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByRole('alert')).toHaveTextContent(
      'The server did not answer',
    )
  },
}

import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import type { AcceptInvitationClient, RequestStatus } from '../api/invitations'
import { createAcceptanceView } from '../model/acceptance'
import AcceptInvitationDialog from './AcceptInvitationDialog.vue'
import { decorateWithFrame } from './testing/storyFrame'
import '../styles.css'

const LINK = 'numen-share:eyJ2IjoxfQ'
const PREVIEW = {
  folder: 'Recipes',
  ownerFingerprint: '0F3A 9C21 77B0 D4E6',
  expiresAt: '2026-11-25T12:00:00Z',
  isReadOnly: true,
}

interface Setup {
  readonly status?: Partial<RequestStatus>
  readonly previewFailure?: string
}

const createClient = (setup: Setup): AcceptInvitationClient => ({
  readSharePreview: () =>
    setup.previewFailure === undefined
      ? Promise.resolve(PREVIEW)
      : Promise.reject(new Error(setup.previewFailure)),
  acceptShareInvitation: () =>
    Promise.resolve({ requestId: 'r1', folder: 'Recipes', ownerFingerprint: PREVIEW.ownerFingerprint }),
  getShareInvitationStatus: () =>
    Promise.resolve({
      phase: 'connecting',
      folder: 'Recipes',
      ownerFingerprint: PREVIEW.ownerFingerprint,
      expiresAt: PREVIEW.expiresAt,
      ...setup.status,
    }),
})

type Step = 'empty' | 'pasted' | 'accepted'

const createView = (setup: Setup, step: Step) => {
  const view = createAcceptanceView({
    client: createClient(setup),
    close: () => undefined,
    refreshMs: 1_000_000,
  })
  if (step !== 'empty') {
    view
      .setLink(LINK)
      .then(() => (step === 'accepted' ? view.accept() : undefined))
      .catch(view.fail)
  }
  return view
}

const meta = {
  title: 'NumenSync/AcceptInvitationDialog',
  component: AcceptInvitationDialog,
  decorators: [decorateWithFrame],
  render: (args) => ({
    components: { AcceptInvitationDialog },
    setup: () => ({ args }),
    template: '<AcceptInvitationDialog v-bind="args" />',
  }),
} satisfies Meta<typeof AcceptInvitationDialog>

export default meta
type Story = StoryObj<typeof meta>

const body = (canvasElement: HTMLElement) => within(canvasElement.ownerDocument.body)

export const Empty: Story = {
  args: { view: createView({}, 'empty') },
  play: async ({ canvasElement }) => {
    const screen = body(canvasElement)
    await expect(await screen.findByRole('textbox', { name: 'Invitation link' })).toHaveValue('')
    await expect(screen.getByRole('button', { name: 'Join' })).toBeDisabled()
  },
}

export const LinkOffer: Story = {
  args: { view: createView({}, 'pasted') },
  play: async ({ canvasElement }) => {
    const screen = body(canvasElement)
    await expect(await screen.findByText(/invited you to Recipes. Read only\./)).toBeVisible()
    await expect(screen.getByText(/0F3A 9C21 77B0… invited you/)).toBeVisible()
    await expect(screen.getByRole('button', { name: 'Accept' })).toBeEnabled()
  },
}

export const BadLink: Story = {
  args: { view: createView({ previewFailure: 'This is not a link to a shared folder' }, 'pasted') },
  play: async ({ canvasElement }) => {
    await expect(await body(canvasElement).findByRole('alert')).toHaveTextContent(
      'This is not a link to a shared folder',
    )
  },
}

export const Connecting: Story = {
  args: { view: createView({}, 'accepted') },
  play: async ({ canvasElement }) => {
    await expect(await body(canvasElement).findByText('Connecting to the owner…')).toBeVisible()
  },
}

export const WaitingForOwner: Story = {
  args: { view: createView({ status: { phase: 'waiting', code: '482913' } }, 'accepted') },
  play: async ({ canvasElement }) => {
    const screen = body(canvasElement)
    await expect(await screen.findByText('482 913')).toBeVisible()
    await expect(screen.getByText('Waiting for the owner to allow')).toBeVisible()
  },
}

export const Received: Story = {
  args: { view: createView({ status: { phase: 'received' } }, 'accepted') },
  play: async ({ canvasElement }) => {
    await expect(await body(canvasElement).findByText(/The folder is shared with you/)).toBeVisible()
  },
}

export const Declined: Story = {
  args: { view: createView({ status: { phase: 'declined' } }, 'accepted') },
  play: async ({ canvasElement }) => {
    await expect(await body(canvasElement).findByText(/The owner declined/)).toBeVisible()
  },
}

export const Expired: Story = {
  args: { view: createView({ status: { phase: 'expired' } }, 'accepted') },
  play: async ({ canvasElement }) => {
    await expect(await body(canvasElement).findByText(/The link stopped working/)).toBeVisible()
  },
}

export const Used: Story = {
  args: { view: createView({ status: { phase: 'used' } }, 'accepted') },
  play: async ({ canvasElement }) => {
    await expect(await body(canvasElement).findByText(/Someone used this link before/)).toBeVisible()
  },
}

export const Failed: Story = {
  args: {
    view: createView({ status: { phase: 'failed', error: 'The owner is offline' } }, 'accepted'),
  },
  play: async ({ canvasElement }) => {
    await expect(await body(canvasElement).findByText('The owner is offline')).toBeVisible()
  },
}

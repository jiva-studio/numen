/**
 * The identity and devices cards of a device that has an account, in the area of a tab: the actions of each card are
 * in its header, the revoke of another device is in its row, and the questions are the shared dialogs.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import type { DeviceInfo, IdentityInfo } from '@/entities/identity'
import type { DeviceCalls } from '@/features/identity-link'
import IdentityDetails from './IdentityDetails.vue'

interface Knobs {
  devices: readonly DeviceInfo[]
  isAskingDelete: boolean
}

const ok = <T>(value: T) => Promise.resolve({ ok: true as const, value })

const INFO: IdentityInfo = {
  accountFingerprint: '3F2A 91C0 7B15 22D4 90AA 11C3 5E6F 0B87',
  deviceFingerprint: 'D5AF 4DDB 6D92 FA46 0390 D870 1186 EB41',
  deviceName: 'Studio laptop',
  deviceAddedAt: '2026-11-22T12:00:00Z',
  isRevoked: false,
  keyEpoch: 1,
}

const THIS_DEVICE: DeviceInfo = {
  name: 'Studio laptop',
  fingerprint: INFO.deviceFingerprint,
  addedAt: INFO.deviceAddedAt,
  isThis: true,
  isRevoked: false,
}

const PHONE: DeviceInfo = {
  name: 'Phone',
  fingerprint: '1111 2222 3333 4444 5555 6666 7777 8888',
  addedAt: '2026-11-23T09:30:00Z',
  isThis: false,
  isRevoked: false,
}

const createCalls = (devices: readonly DeviceInfo[]): DeviceCalls => ({
  list: ref(devices),
  load: fn(() => Promise.resolve()),
  revoke: fn(() => ok(undefined)),
  link: {
    start: fn(() => ok({ ...createLinkInfo(), phase: 'waitingForChannel' as const })),
    get: fn(() => ok({ ...createLinkInfo(), phase: 'waitingForChannel' as const })),
    decide: fn(() => ok(undefined)),
    confirmJoin: fn(() => ok(undefined)),
    cancel: fn(() => ok(undefined)),
  },
})

const createLinkInfo = () => ({
  phase: 'waitingForPeer' as const,
  mode: 'show' as const,
  role: 'host' as const,
  qrText: 'numen://link?v=1',
  peerName: '',
  peerFingerprint: '',
  accountFingerprint: INFO.accountFingerprint,
  reason: undefined,
  deviceName: INFO.deviceName,
  deviceFingerprint: INFO.deviceFingerprint,
})

const onDelete = fn()

const meta: Meta<Knobs> = {
  title: 'Window/Identity/Devices',
  component: IdentityDetails,
  parameters: { layout: 'fullscreen' },
  args: { devices: [THIS_DEVICE, PHONE], isAskingDelete: false },
  render: (args) => ({
    components: { IdentityDetails },
    setup: () => ({ args, info: INFO, calls: createCalls(args.devices), onDelete }),
    template: `
      <div
        class="numen"
        style="position:relative;contain:strict;box-sizing:border-box;inline-size:760px;block-size:560px;margin:1.5rem auto;border:1px solid var(--numen-rule);background:var(--numen-surface);overflow:auto;padding:1rem;display:flex;flex-direction:column;gap:1rem"
      >
        <IdentityDetails
          :info="info"
          :devices="calls"
          problem=""
          :is-busy="false"
          :is-asking-delete="args.isAskingDelete"
          @ask="onDelete"
          @cancel="onDelete"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** Both cards: Delete in the header of the identity, Link a new device in the header of the devices. */
export const Loaded: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await expect(canvas.getByRole('button', { name: 'Link a new device' })).toBeVisible()
    await expect(canvas.getAllByRole('button', { name: 'Remove' })).toHaveLength(1)
  },
}

/** Only this device is listed. */
export const Alone: Story = {
  args: { devices: [THIS_DEVICE] },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement.ownerDocument.body).queryByRole('button', { name: 'Remove' })).toBeNull()
  },
}

/** The header action opens the link dialog over the tab, and the link waits for the server. */
export const Linking: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await userEvent.click(canvas.getByRole('button', { name: 'Link a new device' }))
    await expect(
      await canvas.findByRole('dialog', { name: 'Link a device' }),
    ).toBeVisible()
    await expect(await canvas.findByText('Scan this code.')).toBeVisible()
  },
}

/** Revoking another device asks first, in the shared dialog. */
export const Revoking: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body)
    await userEvent.click(canvas.getByRole('button', { name: 'Remove' }))
    await expect(await canvas.findByRole('alertdialog', { name: 'Remove this device?' })).toBeVisible()
  },
}

/** Deleting the identity asks first, in the shared dialog. */
export const Deleting: Story = {
  args: { isAskingDelete: true },
  play: async ({ canvasElement }) => {
    await expect(
      await within(canvasElement.ownerDocument.body).findByRole('alertdialog', { name: 'Delete the identity from this device?' }),
    ).toBeVisible()
  },
}

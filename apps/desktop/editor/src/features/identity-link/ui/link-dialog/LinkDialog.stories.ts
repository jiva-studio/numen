/**
 * The dialog of a device that has the account and links a new one: it shows its QR code at once,
 * says what it waits for, shows the new device with the confirmation code in large digits, and ends in a clear state.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import type { LinkCalls } from '../../lib/types'
import LinkDialog from './LinkDialog.vue'
import { createCalls, createInfo, createRefusal, WindowFrame } from '../../testing/linkStory'

interface Knobs {
  calls: LinkCalls
  stallMs: number
  onClose: () => void
}

const meta: Meta<Knobs> = {
  title: 'Window/Identity/Link dialog',
  component: LinkDialog,
  parameters: { layout: 'fullscreen' },
  args: { calls: createCalls(createInfo()), stallMs: 15_000, onClose: fn() },
  render: (args) => ({
    components: { LinkDialog, WindowFrame },
    setup: () => ({ args }),
    template: `
      <WindowFrame>
        <LinkDialog
          :calls="args.calls"
          :stall-ms="args.stallMs"
          @close="args.onClose"
        />
      </WindowFrame>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const peer = (overrides: Parameters<typeof createInfo>[0] = {}) =>
  createInfo({
    phase: 'confirmPeer',
    mode: 'show',
    peerName: 'Phone',
    peerFingerprint: '1111 2222 3333 4444 5555 6666 7777 8888',
    code: '482913',
    ...overrides,
  })

/** The QR code, the steps, the Copy link button and what is waited for. */
export const ShowingQr: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByRole('dialog', { name: 'Link a device' }),
    ).toBeVisible()
    await expect(canvas.getByRole('img', { name: 'QR code' })).toBeVisible()
    await expect(canvas.getByText('Waiting for the other device to scan')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Copy link' })).toBeVisible()
  },
}

/** The server has not made the channel yet: the box of the QR code holds a spinner and the line says what happens. */
export const Preparing: Story = {
  args: { calls: createCalls(createInfo({ phase: 'waitingForChannel', qrText: '' })) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Preparing the link')).toBeVisible()
  },
}

/** The server does not answer: the cause, and Try again starts the link again. */
export const Timeout: Story = {
  args: {
    stallMs: 1000,
    calls: createCalls(createInfo({ phase: 'waitingForChannel', qrText: '' })),
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByText(/Numen Sync cannot reach the server/, undefined, { timeout: 8000 }),
    ).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Try again' }))
    await waitFor(() => expect(args.calls.start).toHaveBeenCalledTimes(2))
  },
}

/** The start was refused: the reason and Try again. */
export const Refused: Story = {
  args: { calls: createRefusal('Another link is running.') },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent('Another link is running.')
  },
}


/** The new device is asked about with its name, its fingerprint and the code to type on it. */
export const NewDeviceAsked: Story = {
  args: { calls: createCalls(peer()) },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Phone')).toBeVisible()
    await expect(canvas.getByLabelText('Confirmation code')).toHaveTextContent('482 913')
    await userEvent.click(canvas.getByRole('button', { name: 'Link device' }))
    await expect(args.calls.decide).toHaveBeenCalledWith(true)
  },
}

/** The new device was linked on this side and waits for its code. */
export const WaitingForTheCode: Story = {
  args: { calls: createCalls(peer({ isApproved: true })) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Waiting for the new device')).toBeVisible()
    await expect(canvas.getByLabelText('Confirmation code')).toHaveTextContent('482 913')
  },
}

/** Declining the new device. */
export const Declined: Story = {
  args: { calls: createCalls(peer()) },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: "Don't link" }))
    await expect(args.calls.decide).toHaveBeenCalledWith(false)
  },
}

/** The device is linked. */
export const Linked: Story = {
  args: { calls: createCalls(createInfo({ phase: 'linked' })) },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('The device is linked')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Done' }))
    await expect(args.onClose).toHaveBeenCalled()
  },
}

/** Three wrong codes close the link. */
export const Failed: Story = {
  args: { calls: createCalls(createInfo({ phase: 'failed', reason: 'tooManyWrongCodes' })) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('alert')).toHaveTextContent('Too many wrong codes.')
  },
}

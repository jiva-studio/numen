/**
 * The dialog of a new device that joins an account, : it opens the camera for the QR code of the
 * other device, shows a field for a pasted link only without a camera, and walks through the waiting, the code and the end.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import type { ScannerCamera } from '@numen/ui'
import type { LinkCalls } from '../../lib/types'
import JoinDialog from './JoinDialog.vue'
import {
  createCalls,
  createInfo,
  createRefusal,
  createCodeCamera,
  QR_TEXT,
  WindowFrame,
} from '../../testing/linkStory'

interface Knobs {
  calls: LinkCalls
  findCamera: () => Promise<boolean>
  camera: ScannerCamera | undefined
  stallMs: number
  onClose: () => void
}

const meta: Meta<Knobs> = {
  title: 'Window/Identity/Join dialog',
  component: JoinDialog,
  parameters: { layout: 'fullscreen' },
  args: {
    calls: createCalls(createInfo({ role: 'joiner', phase: 'waitingForAccept', mode: 'scan' })),
    findCamera: () => Promise.resolve(false),
    camera: undefined,
    stallMs: 15_000,
    onClose: fn(),
  },
  render: (args) => ({
    components: { JoinDialog, WindowFrame },
    setup: () => ({ args }),
    template: `
      <WindowFrame>
        <JoinDialog
          :calls="args.calls"
          :find-camera="args.findCamera"
          :stall-ms="args.stallMs"
          v-bind="args.camera ? { camera: args.camera } : {}"
          @close="args.onClose"
        />
      </WindowFrame>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const joiner = (overrides: Parameters<typeof createInfo>[0] = {}) =>
  createInfo({ role: 'joiner', mode: 'scan', accountFingerprint: '', ...overrides })

const submitText = async (canvas: ReturnType<typeof within>, text: string) => {
  await userEvent.click(await canvas.findByRole('textbox', { name: 'Link from the other device' }))
  await userEvent.paste(text)
  await userEvent.click(canvas.getByRole('button', { name: 'Use link' }))
}

/** The camera is being asked for: a spinner says so. */
export const CheckingCamera: Story = {
  args: { findCamera: () => new Promise<boolean>(() => undefined) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dialog = canvas.getByRole('dialog', { name: 'Use existing account' })
    await expect(dialog).toBeVisible()
    await expect(canvas.getByText('Checking the camera')).toBeVisible()
  },
}

/** The camera shows its picture in a frame and the QR code of the other device is read; the core has not answered yet, so the scanner stays. */
export const CameraReady: Story = {
  args: {
    findCamera: () => Promise.resolve(true),
    camera: createCodeCamera(QR_TEXT),
    stallMs: 600_000,
    calls: {
      ...createCalls(joiner()),
      start: fn(() => new Promise<never>(() => undefined)),
    },
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByText('Point the camera at the code on your other device.'),
    ).toBeVisible()
    await expect(canvas.queryByRole('textbox')).toBeNull()
    await expect(canvas.queryByRole('button', { name: /paste/i })).toBeNull()
    await waitFor(() => expect(args.calls.start).toHaveBeenCalledWith('joiner', 'scan', QR_TEXT), {
      timeout: 10_000,
    })
  },
}

/** No camera on the machine: the field for the link text is asked for. */
export const NoCamera: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByText('No camera found. Paste the link from the other device.'),
    ).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Use link' })).toBeDisabled()
  },
}

/** The person did not allow the camera: the reason and the field. */
export const CameraDenied: Story = {
  args: {
    findCamera: () => Promise.resolve(true),
    camera: () => Promise.reject(new DOMException('no', 'NotAllowedError')),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByText('The camera is not allowed. Paste the link from the other device.'),
    ).toBeVisible()
  },
}

/** The link text is pasted and used. */
export const Filled: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(args.calls.start).toHaveBeenCalledWith('joiner', 'scan', QR_TEXT)
  },
}

/** The core refuses the text: the reason stays under the field. */
export const Refused: Story = {
  args: { calls: createRefusal('This text is not the link of a QR code.') },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, 'nonsense')
    await expect(await canvas.findByRole('alert')).toHaveTextContent('not the link of a QR code')
  },
}

/** The link text is accepted and the other device has to answer. */
export const WaitingForTheOtherDevice: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(await canvas.findByText('Waiting for the other device')).toBeVisible()
    await expect(canvas.getByText('Studio laptop')).toBeVisible()
  },
}

/** The other device shows six digits: they are typed here. */
export const EnterCode: Story = {
  args: { calls: createCalls(joiner({ phase: 'enterCode', triesLeft: 2 })) },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(await canvas.findByText('2 tries left.')).toBeVisible()
    await userEvent.type(canvas.getByRole('textbox', { name: /Enter the code/ }), '482913')
    await userEvent.click(canvas.getByRole('button', { name: 'Confirm' }))
    await expect(args.calls.enterCode).toHaveBeenCalledWith('482913')
  },
}

/** The account is offered and joined. */
export const AskedToJoin: Story = {
  args: {
    calls: createCalls(joiner({ phase: 'confirmAccount', accountFingerprint: '3F2A 91C0' })),
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(await canvas.findByText('Join account 3F2A 91C0?')).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Join' }))
    await expect(args.calls.confirmJoin).toHaveBeenCalledWith(true)
  },
}

/** The device has the account. */
export const Linked: Story = {
  args: { calls: createCalls(joiner({ phase: 'linked' })) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(await canvas.findByText('The device is linked')).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Done' })).toBeVisible()
  },
}

/** The other device declined. */
export const Failed: Story = {
  args: { calls: createCalls(joiner({ phase: 'failed', reason: 'declined' })) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(await canvas.findByRole('alert')).toHaveTextContent('The new device was declined.')
    await expect(canvas.getByRole('button', { name: 'Try again' })).toBeVisible()
  },
}

/** The link is sent and the server has not made the channel yet. */
export const Preparing: Story = {
  args: { calls: createCalls(joiner({ phase: 'waitingForChannel', qrText: '' })) },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(await canvas.findByText('Preparing the link')).toBeVisible()
  },
}

/** The server does not answer: the link is given up with the cause and Try again. */
export const Timeout: Story = {
  args: { stallMs: 1000, calls: createCalls(joiner({ phase: 'waitingForChannel', qrText: '' })) },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await submitText(canvas, QR_TEXT)
    await expect(
      await canvas.findByText(/Numen Sync cannot reach the server/, undefined, { timeout: 8000 }),
    ).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Try again' }))
    await waitFor(() => expect(args.calls.start).toHaveBeenCalledTimes(2))
  },
}

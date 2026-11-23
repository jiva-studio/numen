/**
 * The camera of a device read for the QR code of a link. The stories open no camera: each hands the scanner the answer
 * a camera gives.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import QrScanner from './QrScanner.vue'

const meta = {
  title: 'QrScanner',
  component: QrScanner,
  parameters: { layout: 'centered' },
} satisfies Meta<typeof QrScanner>

export default meta
type Story = StoryObj<typeof meta>

/** The camera has not answered: the person has not chosen yet in the question of the machine. */
export const WaitingForPermission: Story = {
  args: { camera: () => new Promise<MediaStream>(() => undefined) },
}

/** The person said no. */
export const Denied: Story = {
  args: { camera: () => Promise.reject(new DOMException('no', 'NotAllowedError')) },
}

/** The machine has no camera. */
export const NoCamera: Story = {
  args: { camera: () => Promise.reject(new DOMException('none', 'NotFoundError')) },
}

/** A camera that shows a picture of no code. */
export const Scanning: Story = {
  args: {
    camera: () => {
      const canvas = document.createElement('canvas')
      canvas.width = 320
      canvas.height = 240
      const drawing = canvas.getContext('2d')
      if (drawing !== null) {
        drawing.fillStyle = '#888888'
        drawing.fillRect(0, 0, canvas.width, canvas.height)
      }
      return Promise.resolve(canvas.captureStream(8))
    },
  },
}

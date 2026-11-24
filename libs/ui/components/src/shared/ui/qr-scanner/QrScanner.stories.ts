/**
 * The camera of a device read for the QR code of a link. The stories open no camera: each hands the scanner the answer
 * a camera gives.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { encode } from 'uqr'
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

const LINK_TEXT =
  'numen://link?v=1&c=AAAAAAAAAAAAAAAAAAAAAA&k=BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB&s=CCCCCCCCCCCCCCCCCCCCCC'
const CELL = 8
const QUIET = 4
const REFRESH_MS = 100
const FRAME_RATE = 8

const filmCode = (text: string) => (): Promise<MediaStream> => {
  const { data: modules } = encode(text, { ecc: 'M', border: 0 })
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = (modules.length + 2 * QUIET) * CELL
  const drawing = canvas.getContext('2d')
  const paint = (isEven: boolean): void => {
    if (drawing === null) return
    drawing.fillStyle = '#ffffff'
    drawing.fillRect(0, 0, canvas.width, canvas.height)
    drawing.fillStyle = '#000000'
    modules.forEach((row, y) =>
      row.forEach((isDark, x) => {
        if (isDark) drawing.fillRect((x + QUIET) * CELL, (y + QUIET) * CELL, CELL, CELL)
      }),
    )
    drawing.fillStyle = isEven ? '#fefefe' : '#ffffff'
    drawing.fillRect(0, 0, 1, 1)
  }
  paint(true)
  let isEven = false
  const stream = canvas.captureStream(FRAME_RATE)
  const refresh = setInterval(() => paint((isEven = !isEven)), REFRESH_MS)
  stream
    .getTracks()
    .forEach((track) => track.addEventListener('ended', () => clearInterval(refresh)))
  return Promise.resolve(stream)
}

/** The camera films the QR code of a link: the scanner says its text once. */
export const ReadsALink: Story = {
  args: { camera: filmCode(LINK_TEXT), onScanned: fn(), onFailed: fn() },
  play: async ({ args }) => {
    await waitFor(() => expect(args.onScanned).toHaveBeenCalledWith(LINK_TEXT), { timeout: 10_000 })
    await expect(args.onFailed).not.toHaveBeenCalled()
  },
}

/** The camera films a QR code that is not a link of Numen: the scanner says so and gives no text. */
export const ShowsAnInvalidCode: Story = {
  args: {
    camera: filmCode('https://example.org/menu'),
    isAccepted: (text: string) => text.startsWith('numen://link'),
    onScanned: fn(),
    onFailed: fn(),
  },
  play: async ({ args, canvasElement }) => {
    const alert = await within(canvasElement).findByRole('alert', undefined, { timeout: 10_000 })
    await expect(alert).toHaveTextContent('This code is not a link of Numen.')
    await expect(args.onFailed).toHaveBeenCalledWith('This code is not a link of Numen.')
    await expect(args.onScanned).not.toHaveBeenCalled()
  },
}

/** The first answer of the camera is no, the person asks again and the camera films the QR code of a link. */
export const TriesAgainAfterAFailure: Story = {
  args: {
    camera: (() => {
      let isFirst = true
      return () => {
        if (!isFirst) return filmCode(LINK_TEXT)()
        isFirst = false
        return Promise.reject(new DOMException('no', 'NotAllowedError'))
      }
    })(),
    onScanned: fn(),
    onFailed: fn(),
  },
  play: async ({ args, canvasElement }) => {
    const retry = await within(canvasElement).findByRole('button', { name: 'Try again' })
    await expect(args.onFailed).toHaveBeenCalledWith('The camera is not allowed.')
    await userEvent.click(retry)
    await waitFor(() => expect(args.onScanned).toHaveBeenCalledWith(LINK_TEXT), { timeout: 10_000 })
  },
}

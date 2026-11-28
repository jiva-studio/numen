/** Reads the QR code a page renders: a picture of it is decoded with jsQR in a blank page of the same browser. */
import { createRequire } from 'node:module'
import type { BrowserContext } from 'playwright/test'

const PADDING = 40
const require = createRequire(import.meta.url)

declare global {
  interface Window {
    jsQR(data: Uint8ClampedArray, width: number, height: number): { data: string } | null
  }
}

/** The text of the QR code in a PNG of it, with a margin of white around. It throws when no code is in the picture. */
export const decodeQr = async (context: BrowserContext, png: Buffer): Promise<string> => {
  const blank = await context.newPage()
  try {
    await blank.goto('about:blank')
    await blank.addScriptTag({ path: require.resolve('jsqr') })
    const decoded = await blank.evaluate(
      async ({ url, padding }) => {
        const image = new Image()
        image.src = url
        await image.decode()
        const canvas = document.createElement('canvas')
        canvas.width = image.width + 2 * padding
        canvas.height = image.height + 2 * padding
        const drawing = canvas.getContext('2d')
        if (drawing === null) return ''
        drawing.fillStyle = '#ffffff'
        drawing.fillRect(0, 0, canvas.width, canvas.height)
        drawing.drawImage(image, padding, padding)
        const pixels = drawing.getImageData(0, 0, canvas.width, canvas.height)
        return window.jsQR(pixels.data, pixels.width, pixels.height)?.data ?? ''
      },
      { url: `data:image/png;base64,${png.toString('base64')}`, padding: PADDING },
    )
    if (decoded === '') throw new Error('the picture of the QR code holds no code that jsQR reads')
    return decoded
  } finally {
    await blank.close()
  }
}

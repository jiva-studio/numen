/** A QR code is drawn from its text and read back to the same text. */
import { mount } from '@vue/test-utils'
import jsQR from 'jsqr'
import { describe, expect, it } from 'vitest'
import QrCode from './QrCode.vue'

const TEXT = 'numen://link?v=1&c=AAAAAAAAAAAAAAAAAAAAAA&k=BBBB&s=CCCC'
const SCALE = 6

const readPixels = (wrapper: ReturnType<typeof mount>): { data: Uint8ClampedArray; side: number } => {
  const box = wrapper.get('svg').attributes('viewBox')?.split(' ') ?? []
  const span = Number(box[2])
  const side = span * SCALE
  const data = new Uint8ClampedArray(side * side * 4).fill(255)
  const path = wrapper.get('path').attributes('d') ?? ''
  for (const [, x, y, width] of path.matchAll(/M(\d+) (\d+)h(\d+)v1h-\d+z/g)) {
    for (let row = Number(y) * SCALE; row < (Number(y) + 1) * SCALE; row++) {
      for (let column = Number(x) * SCALE; column < (Number(x) + Number(width)) * SCALE; column++) {
        data.fill(0, (row * side + column) * 4, (row * side + column) * 4 + 3)
      }
    }
  }
  return { data, side }
}

describe('a QR code', () => {
  it('is a picture named QR code', () => {
    const wrapper = mount(QrCode, { props: { text: TEXT } })
    expect(wrapper.get('svg').attributes('role')).toBe('img')
    expect(wrapper.get('svg').attributes('aria-label')).toBe('QR code')
  })

  it('holds the text it was given', () => {
    const { data, side } = readPixels(mount(QrCode, { props: { text: TEXT } }))
    expect(jsQR(data, side, side)?.data).toBe(TEXT)
  })

  it('draws another text as another picture', () => {
    const first = mount(QrCode, { props: { text: TEXT } }).get('path').attributes('d')
    const second = mount(QrCode, { props: { text: `${TEXT}x` } }).get('path').attributes('d')
    expect(first).not.toBe(second)
  })

  it('takes the size it is given', () => {
    const wrapper = mount(QrCode, { props: { text: TEXT, size: 120 } })
    expect(wrapper.get('svg').attributes('width')).toBe('120')
  })
})

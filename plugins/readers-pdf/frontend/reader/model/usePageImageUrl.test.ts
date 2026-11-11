import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { usePageImageUrl } from './usePageImageUrl'

const createUrl = (width: number, sharp: readonly number[]) =>
  usePageImageUrl({
    getPicture: () => (page) => `/page/${page}.png`,
    drawnWidth: ref(width),
    sharpPages: ref(new Set(sharp)),
  })

describe('the picture address of a page', () => {
  it('is the address the caller gives for a sharp page', () => {
    expect(createUrl(800, [1, 2])(2)).toBe('/page/2.png')
  })

  it('is empty for a page that is not sharp', () => {
    expect(createUrl(800, [1, 2])(5)).toBe('')
  })

  it('is empty while no width is known', () => {
    expect(createUrl(0, [1, 2])(2)).toBe('')
  })
})

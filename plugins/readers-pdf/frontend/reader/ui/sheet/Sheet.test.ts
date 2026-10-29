import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Sheet from './Sheet.vue'

const HIGHLIGHTS = [
  { minX: 0.1, minY: 0.2, maxX: 0.6, maxY: 0.26 },
  { minX: 0.1, minY: 0.27, maxX: 0.4, maxY: 0.33 },
]

const sheet = (picture = '/assets/book/pages/0?wide=400') =>
  mount(Sheet, { props: { at: 0, picture, highlights: HIGHLIGHTS } })

describe('a page still coming', () => {
  it('paints nothing of the picture', () => {
    // An `img` pointing at an image that has not loaded yet shows the browser's
    // broken-image mark and its alt text, rendered in the shape of an undrawn
    // page.
    const page = sheet()

    expect(page.find('.reader__picture').classes()).toContain('invisible')
  })

  it('turns a ring where the picture will be', () => {
    const page = sheet()

    expect(page.find('.reader__spinner').exists()).toBe(true)
    expect(page.find('.spinner').exists()).toBe(true)
  })

  it('lights nothing', () => {
    // A rectangle over a page still coming is a mark on nothing, standing where
    // the page is not.
    expect(sheet().findAll('.reader__highlight')).toHaveLength(0)
  })
})

describe('a page that has come', () => {
  const loadPage = async () => {
    const page = sheet()
    await page.find('.reader__picture').trigger('load')
    return page
  }

  it('paints the picture', async () => {
    expect((await loadPage()).find('.reader__picture').classes()).not.toContain('invisible')
  })

  it('takes the ring away', async () => {
    expect((await loadPage()).find('.reader__spinner').exists()).toBe(false)
  })

  it('lights what was found on it', async () => {
    expect((await loadPage()).findAll('.reader__highlight')).toHaveLength(HIGHLIGHTS.length)
  })

  it('marks the other places apart from the one it was opened at', async () => {
    // Two ways of rendering one page: the location the user was sent to, and the
    // others, which indicate content is present but are not the target.
    const otherHighlights = [{ minX: 0.1, minY: 0.6, maxX: 0.5, maxY: 0.66 }]
    const page = mount(Sheet, {
      props: {
        at: 0,
        picture: '/assets/book/pages/0?wide=400',
        highlights: HIGHLIGHTS,
        otherHighlights,
      },
    })
    await page.find('.reader__picture').trigger('load')

    expect(page.findAll('.reader__other-highlight')).toHaveLength(otherHighlights.length)
    expect(page.findAll('.reader__highlight')).toHaveLength(HIGHLIGHTS.length)
  })
})

describe('a page that will not come', () => {
  const mountRefusedPage = async () => {
    const page = sheet()
    for (let ask = 0; ask <= 4; ask++) {
      await page.find('.reader__picture').trigger('error')
      if (!page.find('.reader__picture').exists()) break
    }
    return page
  }

  it('says so, once it has been requested for enough times', async () => {
    const page = await mountRefusedPage()

    expect(page.text()).toContain('Failed to load page')
    expect(page.find('.spinner').exists()).toBe(false)
  })

  it('asks again at an address the last ask did not use', async () => {
    // again, so each attempt has to differ.
    const page = sheet()
    const first = page.find('.reader__picture').attributes('src')

    await page.find('.reader__picture').trigger('error')

    expect(page.find('.reader__picture').attributes('src')).not.toBe(first)
  })

  it('stops asking, and does not point at a picture at all', async () => {
    expect((await mountRefusedPage()).find('.reader__picture').exists()).toBe(false)
  })
})

describe('a page pointed somewhere else', () => {
  it('is requested for afresh', async () => {
    // Reset retry state when the page image URL changes.
    const page = sheet()
    await page.find('.reader__picture').trigger('error')
    await page.find('.reader__picture').trigger('load')

    await page.setProps({ picture: '/assets/book/pages/0?wide=800' })

    expect(page.find('.reader__picture').attributes('src')).toBe('/assets/book/pages/0?wide=800')
    expect(page.find('.reader__picture').classes()).toContain('invisible')
  })
})

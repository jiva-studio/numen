import { mount, type VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Sheet from './Sheet.vue'

const HIGHLIGHTS = [
  { minX: 0.1, minY: 0.2, maxX: 0.6, maxY: 0.26 },
  { minX: 0.1, minY: 0.27, maxX: 0.4, maxY: 0.33 },
]

const SMALL = '/assets/book/pages/0?wide=400'
const LARGE = '/assets/book/pages/0?wide=800'

const sheet = (picture = SMALL) =>
  mount(Sheet, { props: { at: 0, picture, highlights: HIGHLIGHTS } })

const incoming = (page: VueWrapper) => page.find('.reader__incoming')
const getPicture = (page: VueWrapper) => page.find('.reader__picture')

describe('a page still coming', () => {
  it('shows no picture', () => {
    // An `img` pointing at an image that has not loaded yet shows the browser's
    // broken-image mark and its alt text, rendered in the shape of an undrawn
    // page.
    const page = sheet()

    expect(getPicture(page).exists()).toBe(false)
    expect(incoming(page).attributes('src')).toBe(SMALL)
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
    await incoming(page).trigger('load')
    return page
  }

  it('shows the picture', async () => {
    expect(getPicture(await loadPage()).attributes('src')).toBe(SMALL)
  })

  it('stops loading it again', async () => {
    expect(incoming(await loadPage()).attributes('src')).toBeUndefined()
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
      props: { at: 0, picture: SMALL, highlights: HIGHLIGHTS, otherHighlights },
    })
    await incoming(page).trigger('load')

    expect(page.findAll('.reader__other-highlight')).toHaveLength(otherHighlights.length)
    expect(page.findAll('.reader__highlight')).toHaveLength(HIGHLIGHTS.length)
  })
})

describe('a page that will not come', () => {
  const mountRefusedPage = async () => {
    const page = sheet()
    for (let ask = 0; ask <= 4; ask++) {
      if (incoming(page).attributes('src') === undefined) break
      await incoming(page).trigger('error')
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
    const first = incoming(page).attributes('src')

    await incoming(page).trigger('error')

    expect(incoming(page).attributes('src')).not.toBe(first)
    expect(incoming(page).attributes('src')).toContain('again=1')
  })

  it('stops asking', async () => {
    expect(incoming(await mountRefusedPage()).attributes('src')).toBeUndefined()
  })
})

describe('a page pointed somewhere else', () => {
  it('is requested for afresh', async () => {
    // Reset retry state when the page image URL changes.
    const page = sheet()
    await incoming(page).trigger('error')

    await page.setProps({ picture: LARGE })

    expect(incoming(page).attributes('src')).toBe(LARGE)
  })

  it('keeps showing the picture it has until the new one has loaded', async () => {
    const page = sheet()
    await incoming(page).trigger('load')

    await page.setProps({ picture: LARGE })

    expect(getPicture(page).attributes('src')).toBe(SMALL)
    expect(page.find('.reader__spinner').exists()).toBe(false)
    expect(incoming(page).attributes('src')).toBe(LARGE)
  })

  it('shows the new picture once it has loaded', async () => {
    const page = sheet()
    await incoming(page).trigger('load')
    await page.setProps({ picture: LARGE })

    await incoming(page).trigger('load')

    expect(getPicture(page).attributes('src')).toBe(LARGE)
    expect(incoming(page).attributes('src')).toBeUndefined()
  })

  it('drops a request in flight when a newer picture is wanted', async () => {
    const page = sheet()
    await incoming(page).trigger('load')
    await page.setProps({ picture: LARGE })

    await page.setProps({ picture: '/assets/book/pages/0?wide=1024' })

    expect(incoming(page).attributes('src')).toBe('/assets/book/pages/0?wide=1024')
  })

  it('drops the request and keeps the picture when nothing is wanted', async () => {
    const page = sheet()
    await incoming(page).trigger('load')
    await page.setProps({ picture: LARGE })

    await page.setProps({ picture: '' })

    expect(incoming(page).attributes('src')).toBeUndefined()
    expect(getPicture(page).attributes('src')).toBe(SMALL)
  })

  it('does not load the picture it shows again', async () => {
    const page = sheet()
    await incoming(page).trigger('load')
    await page.setProps({ picture: LARGE })

    await page.setProps({ picture: SMALL })

    expect(incoming(page).attributes('src')).toBeUndefined()
  })

  it('cancels the request in flight when it is removed', () => {
    const page = sheet()
    const image = incoming(page).element

    page.unmount()

    expect(image.hasAttribute('src')).toBe(false)
  })
})

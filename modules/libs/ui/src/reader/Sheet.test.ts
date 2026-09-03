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
    // An `img` pointed at a picture that is not there yet is the browser's own
    // broken-picture mark and the words standing in for it, drawn in the shape
    // of a page nobody has drawn.
    const page = sheet()

    expect(page.find('.reader__picture').classes()).toContain('invisible')
  })

  it('turns a ring where the picture will be', () => {
    const page = sheet()

    expect(page.find('.reader__waiting').exists()).toBe(true)
    expect(page.find('.waiting').exists()).toBe(true)
  })

  it('lights nothing', () => {
    // A rectangle over a page still coming is a mark on nothing, standing where
    // the page is not.
    expect(sheet().findAll('.reader__highlight')).toHaveLength(0)
  })
})

describe('a page that has come', () => {
  const arrived = async () => {
    const page = sheet()
    await page.find('.reader__picture').trigger('load')
    return page
  }

  it('paints the picture', async () => {
    expect((await arrived()).find('.reader__picture').classes()).not.toContain('invisible')
  })

  it('takes the ring away', async () => {
    expect((await arrived()).find('.reader__waiting').exists()).toBe(false)
  })

  it('lights what was found on it', async () => {
    expect((await arrived()).findAll('.reader__highlight')).toHaveLength(HIGHLIGHTS.length)
  })

  it('marks the other places apart from the one it was opened at', async () => {
    // Two ways of drawing one page: the place the person was sent to, and the
    // others, which say there is something here and are not where they are.
    const also = [{ minX: 0.1, minY: 0.6, maxX: 0.5, maxY: 0.66 }]
    const page = mount(Sheet, {
      props: { at: 0, picture: '/assets/book/pages/0?wide=400', highlights: HIGHLIGHTS, also },
    })
    await page.find('.reader__picture').trigger('load')

    expect(page.findAll('.reader__also')).toHaveLength(also.length)
    expect(page.findAll('.reader__highlight')).toHaveLength(HIGHLIGHTS.length)
  })
})

describe('a page that will not come', () => {
  /** Asked for again as many times as it is going to be, and then refused. */
  const givenUp = async () => {
    const page = sheet()
    for (let ask = 0; ask <= 4; ask++) {
      await page.find('.reader__picture').trigger('error')
      if (!page.find('.reader__picture').exists()) break
    }
    return page
  }

  it('says so, once it has been asked for enough times', async () => {
    const page = await givenUp()

    expect(page.text()).toContain('This page would not come.')
    expect(page.find('.waiting').exists()).toBe(false)
  })

  it('asks again at an address the last ask did not use', async () => {
    // A picture at an address the browser already refused is not asked for
    // again, so each attempt has to differ.
    const page = sheet()
    const first = page.find('.reader__picture').attributes('src')

    await page.find('.reader__picture').trigger('error')

    expect(page.find('.reader__picture').attributes('src')).not.toBe(first)
  })

  it('stops asking, and does not point at a picture at all', async () => {
    expect((await givenUp()).find('.reader__picture').exists()).toBe(false)
  })
})

describe('a page pointed somewhere else', () => {
  it('is asked for afresh', async () => {
    // Another address is another question. What would not come is what would
    // not come at that width.
    const page = sheet()
    await page.find('.reader__picture').trigger('error')
    await page.find('.reader__picture').trigger('load')

    await page.setProps({ picture: '/assets/book/pages/0?wide=800' })

    expect(page.find('.reader__picture').attributes('src')).toBe(
      '/assets/book/pages/0?wide=800',
    )
    expect(page.find('.reader__picture').classes()).toContain('invisible')
  })
})

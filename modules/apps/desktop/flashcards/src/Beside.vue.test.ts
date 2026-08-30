// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'

import Beside from './Beside.vue'

/**
 * The pair on the screen. jsdom lays nothing out, so the strip is given the
 * width it would have had.
 */
const pair = (open: boolean) => {
  const one = mount(Beside, {
    props: { open },
    slots: { default: '<p>the one</p>', other: '<p>the other</p>' },
  })
  const strip = one.find('.beside').element as HTMLElement
  Object.defineProperty(strip, 'scrollWidth', { value: 1000, configurable: true })
  Object.defineProperty(strip, 'clientWidth', { value: 600, configurable: true })
  return { one, strip }
}

describe('one thing and a second beside it', () => {
  it('draws both, whether the second is scrolled to or not', () => {
    for (const open of [false, true]) {
      const { one } = pair(open)
      expect(one.text()).toContain('the one')
      expect(one.text()).toContain('the other')
    }
  })

  // Whichever is out of the window is reached by nothing: not the keyboard, and
  // not what reads the screen aloud.
  it('puts whichever is out of the window beyond reach', () => {
    const shut = pair(false).one
    expect(shut.find('.beside__one').attributes('inert')).toBeUndefined()
    expect(shut.find('.beside__other').attributes('inert')).toBeDefined()

    const open = pair(true).one
    expect(open.find('.beside__one').attributes('inert')).toBeDefined()
    expect(open.find('.beside__other').attributes('inert')).toBeUndefined()
  })

  // Asked for by a key rather than a hand, the strip is taken there rather than
  // put there, so it reads as the same movement either way.
  it('scrolls to the second when it is asked for, and back when it is not', async () => {
    const { one, strip } = pair(false)

    await one.setProps({ open: true })
    expect(strip.scrollLeft).toBe(400)

    await one.setProps({ open: false })
    expect(strip.scrollLeft).toBe(0)
  })

  // A hand takes the strip where it likes, and past the halfway mark it has
  // asked for the second.
  it('says the second is wanted once the strip is past halfway', async () => {
    const { one, strip } = pair(false)

    strip.scrollLeft = 300
    await strip.dispatchEvent(new Event('scroll'))
    await nextTick()

    expect(one.emitted('update:open')).toEqual([[true]])
  })

  // A strip taken somewhere by a key passes the halfway mark on the way, and
  // that is not a person asking for the one it is leaving.
  it('says nothing about where it is passing through on its way', async () => {
    const { one, strip } = pair(true)

    await one.setProps({ open: false })
    strip.scrollLeft = 300
    await strip.dispatchEvent(new Event('scroll'))
    await nextTick()

    expect(one.emitted('update:open')).toBeUndefined()
  })

  it('says nothing while the strip is short of halfway', async () => {
    const { one, strip } = pair(false)

    strip.scrollLeft = 100
    await strip.dispatchEvent(new Event('scroll'))
    await nextTick()

    expect(one.emitted('update:open')).toBeUndefined()
  })
})

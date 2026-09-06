/**
 * What a link note points at, drawn.
 *
 * The frame is the one place this window loads anything from a host that is not
 * its own, so what it is given is worth holding to.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Pointing from './Pointing.vue'
import { WORDS as words } from './words'

const VIDEO = {
  url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
  embed: 'https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ?enablejsapi=1',
}

describe('an address something plays', () => {
  it('is played in a frame, from the host the window may frame', () => {
    const drawn = mount(Pointing, { props: { points: VIDEO, words } })
    const frame = drawn.get('iframe')

    expect(frame.attributes('src')).toBe(
      `${VIDEO.embed}&origin=${encodeURIComponent(window.location.origin)}`,
    )
    expect(frame.attributes('title')).toBe(words.playing)
  })

  it('tells the frame which page holds it, so nothing else can drive it', () => {
    const drawn = mount(Pointing, { props: { points: VIDEO, words } })

    expect(drawn.get('iframe').attributes('src')).toContain('origin=')
  })

  it('is sandboxed, and sends nothing about where the person came from', () => {
    const frame = mount(Pointing, { props: { points: VIDEO, words } }).get('iframe')

    expect(frame.attributes('sandbox')).toBe('allow-scripts allow-same-origin allow-presentation')
    expect(frame.attributes('referrerpolicy')).toBe('no-referrer')
  })
})

describe('an address nothing plays', () => {
  const page = { url: 'https://example.com/a', embed: '' }

  it('is the address itself, and no frame at all', () => {
    const drawn = mount(Pointing, { props: { points: page, words } })

    expect(drawn.find('iframe').exists()).toBe(false)
    expect(drawn.text()).toContain('https://example.com/a')
  })
})

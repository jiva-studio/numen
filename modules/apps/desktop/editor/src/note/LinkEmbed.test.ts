/**
 * What a link note points at, drawn.
 *
 * The frame is the one place this window loads anything from a host that is not
 * its own, so what it is given is worth holding to.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import LinkEmbed from './LinkEmbed.vue'
import { WORDS as words } from './words'

const VIDEO = {
  url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
  embed: 'https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ?enablejsapi=1',
}

describe('an address something plays', () => {
  it('is played in a frame, from the host the window may frame', () => {
    const drawn = mount(LinkEmbed, { props: { address: VIDEO, cues: [], copy: '', words } })
    const frame = drawn.get('iframe')

    expect(frame.attributes('src')).toBe(
      `${VIDEO.embed}&origin=${encodeURIComponent(window.location.origin)}`,
    )
    expect(frame.attributes('title')).toBe(words.playing)
  })

  it('tells the frame which page holds it, so nothing else can drive it', () => {
    const drawn = mount(LinkEmbed, { props: { address: VIDEO, cues: [], copy: '', words } })

    expect(drawn.get('iframe').attributes('src')).toContain('origin=')
  })

  it('is sandboxed, and sends nothing about where the person came from', () => {
    const drawn = mount(LinkEmbed, { props: { address: VIDEO, cues: [], copy: '', words } })
    const frame = drawn.get('iframe')

    expect(frame.attributes('sandbox')).toBe('allow-scripts allow-same-origin allow-presentation')
    expect(frame.attributes('referrerpolicy')).toBe('no-referrer')
  })
})

describe('an address nothing plays', () => {
  const page = { url: 'https://example.com/a', embed: '' }

  it('is the address itself, and no frame at all', () => {
    const drawn = mount(LinkEmbed, { props: { address: page, cues: [], copy: '', words } })

    expect(drawn.find('iframe').exists()).toBe(false)
    expect(drawn.text()).toContain('https://example.com/a')
  })
})

describe('the words fetched for an address', () => {
  const CUES = [
    { text: 'what was said', from: 1_500, to: 4_200 },
    { text: 'what was said next', from: 83_000, to: 85_000 },
  ]

  it('stand under the player, each at the moment it was said', () => {
    const drawn = mount(LinkEmbed, { props: { address: VIDEO, cues: CUES, copy: '', words } })

    const said = drawn.findAll('.embed__said')
    expect(said).toHaveLength(2)
    expect(said[1]?.text()).toContain('what was said next')
    expect(said[1]?.text()).toContain('1:23')
  })

  it('plays the frame from that moment, and tells nobody else', async () => {
    const drawn = mount(LinkEmbed, {
      props: { address: VIDEO, cues: CUES, copy: '', words },
      attachTo: document.body,
    })
    const said: [string, string][] = []
    const frame = drawn.get('iframe').element as HTMLIFrameElement
    Object.defineProperty(frame, 'contentWindow', {
      value: { postMessage: (message: string, origin: string) => said.push([message, origin]) },
    })

    await drawn.findAll('.embed__said')[1]?.trigger('click')

    expect(said).toHaveLength(1)
    expect(JSON.parse(said[0]![0])).toEqual({
      event: 'command',
      func: 'seekTo',
      args: [83, true],
    })
    expect(said[0]![1]).toBe('https://www.youtube-nocookie.com')
  })

  it('are nothing where nothing has been fetched', () => {
    const drawn = mount(LinkEmbed, { props: { address: VIDEO, cues: [], copy: '', words } })

    expect(drawn.find('.embed__words').exists()).toBe(false)
  })
})

describe('a copy of the video on this disk', () => {
  const COPY = 'http://127.0.0.1:9/token/vault/notes%2Ftalk.md?size=12&mtime=0'
  const CUES = [{ text: 'what was said', from: 1_500, to: 4_200 }]

  it('is played in place of the frame, and nothing of the site is loaded', () => {
    const drawn = mount(LinkEmbed, { props: { address: VIDEO, cues: CUES, copy: COPY, words } })

    expect(drawn.find('iframe').exists()).toBe(false)
    expect(drawn.get('video').attributes('src')).toBe(COPY)
  })

  it('plays from the moment a stretch of speech was said', async () => {
    const drawn = mount(LinkEmbed, {
      props: { address: VIDEO, cues: CUES, copy: COPY, words },
      attachTo: document.body,
    })
    const player = drawn.get('video').element as HTMLVideoElement
    player.play = () => Promise.resolve()

    await drawn.get('.embed__said').trigger('click')

    expect(player.currentTime).toBe(1.5)
  })
})

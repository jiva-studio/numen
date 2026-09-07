/**
 * What a url points at, played.
 *
 * The window frames the socket this run opened and no host at all: what it is
 * given is an address on this machine, and the page there is what frames a
 * host. What this window hands that page is worth holding to.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Embed from './Embed.vue'
import { WORDS as words } from '../recording/words'

// The player is framed from the socket this run opened, which is the address a
// host is told is holding it.
const EMBED = 'http://127.0.0.1:9/token/embed/https%3A%2F%2Fwww.youtube.com%2Fwatch%3Fv%3DdQw4w9WgXcQ'

describe('an address something plays', () => {
  // The address is composed where the socket that serves the player is open,
  // and this frames what it was given. A window that composed one of its own
  // would be a second place the answer is decided.
  it('is played in a frame, at the address it was given', () => {
    const drawn = mount(Embed, { props: { embed: EMBED, copy: '', words } })
    const frame = drawn.get('iframe')

    expect(frame.attributes('src')).toBe(EMBED)
    expect(frame.attributes('title')).toBe(words.playing)
  })

  // A frame told to send no address of the page holding it hands that down to
  // the page it loads, and that page has a host to answer to: one told nothing
  // about who is framing it plays nothing. What this window frames is its own
  // socket, so there is nothing here to keep from anybody.
  it('is sandboxed, and tells the page it holds nothing about referrers', () => {
    const drawn = mount(Embed, { props: { embed: EMBED, copy: '', words } })
    const frame = drawn.get('iframe')

    expect(frame.attributes('sandbox')).toBe('allow-scripts allow-same-origin allow-popups')
    expect(frame.attributes('referrerpolicy')).toBeUndefined()
  })
})

describe('an address nothing plays', () => {
  it('is nothing at all: the tab says where it points', () => {
    const drawn = mount(Embed, { props: { embed: '', copy: '', words } })

    expect(drawn.find('iframe').exists()).toBe(false)
    expect(drawn.find('video').exists()).toBe(false)
  })
})

describe('a moment chosen in the transcript', () => {
  it('is where the frame is told to play from, and nobody else is told', () => {
    const drawn = mount(Embed, {
      props: { embed: EMBED, copy: '', words },
      attachTo: document.body,
    })
    const said: [string, string][] = []
    const frame = drawn.get('iframe').element as HTMLIFrameElement
    Object.defineProperty(frame, 'contentWindow', {
      value: { postMessage: (message: string, origin: string) => said.push([message, origin]) },
    })

    drawn.vm.seeks(83_000)

    expect(said).toHaveLength(1)
    expect(JSON.parse(said[0]![0])).toEqual({
      event: 'command',
      func: 'seekTo',
      args: [83, true],
    })
    // The message goes to the page holding the player and to nothing else. That
    // page hands it on to the host, and this window speaks to no host at all.
    expect(said[0]![1]).toBe('http://127.0.0.1:9')
  })

  it('is where the copy is played from, where a copy stands', () => {
    const COPY = 'http://127.0.0.1:9/token/vault/notes%2Ftalk.md?size=12&mtime=0'
    const drawn = mount(Embed, {
      props: { embed: EMBED, copy: COPY, words },
      attachTo: document.body,
    })
    const player = drawn.get('video').element as HTMLVideoElement
    player.play = () => Promise.resolve()

    drawn.vm.seeks(1_500)

    expect(player.currentTime).toBe(1.5)
  })
})

describe('a copy of the video on this disk', () => {
  const COPY = 'http://127.0.0.1:9/token/vault/notes%2Ftalk.md?size=12&mtime=0'

  it('is played in place of the frame, and nothing of the site is loaded', () => {
    const drawn = mount(Embed, { props: { embed: EMBED, copy: COPY, words } })

    expect(drawn.find('iframe').exists()).toBe(false)
    expect(drawn.get('video').attributes('src')).toBe(COPY)
  })
})

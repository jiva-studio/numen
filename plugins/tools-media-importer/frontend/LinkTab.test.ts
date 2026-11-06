import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import LinkTab from './LinkTab.vue'
import type { Links } from './types'

const createLinks = (media: string, text: string, isTextDeleted = false, page = ''): Links => ({
  getLink: async () => ({
    ok: true,
    value: {
      url: 'https://videos.example/clip',
      isCopyable: true,
      isTextDeleted,
      media,
      page,
    },
  }),
  readArticle: async () => text,
  getSource: async (address) =>
    address.endsWith('.html')
      ? {
          mediaUrl: 'http://127.0.0.1:1/t/plugin/v/importer/clip.html?size=1',
          mediaType: 'text/html',
        }
      : {
          mediaUrl: 'http://127.0.0.1:1/t/plugin/v/importer/clip.wav?size=1',
          mediaType: 'audio/wav',
        },
  fetchText: async () => ({ ok: true, value: null }),
  copyVideo: async () => ({ ok: true, value: null }),
  deleteCopy: async () => ({ ok: true, value: null }),
  watchWork: () => () => undefined,
})

describe('LinkTab', () => {
  it('shows the text fetched from the link', async () => {
    const wrapper = mount(LinkTab, {
      props: {
        state: { path: 'links/rance.md' },
        links: createLinks('', 'A tidal mill stored seawater.'),
      },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('A tidal mill stored seawater.')
    expect(wrapper.find('audio').exists()).toBe(false)
    expect(wrapper.find('a').attributes('href')).toBe('https://videos.example/clip')
    expect(wrapper.text()).toContain('Download video')
  })

  it('plays the copy of the video from where the window serves it', async () => {
    const wrapper = mount(LinkTab, {
      props: {
        path: 'links/clip.md',
        links: createLinks('plugin://md.numen.tools.media.importer/0b5e1c.mp4', ''),
      },
    })
    await flushPromises()
    expect(wrapper.find('audio').attributes('src')).toBe(
      'http://127.0.0.1:1/t/plugin/v/importer/clip.wav?size=1',
    )
    expect(wrapper.get('.link-player__button').text()).toBe('Play')
    expect(wrapper.text()).toContain('0:00')
    expect(wrapper.text()).toContain('Delete downloaded video')
  })

  it('says when nothing was fetched', async () => {
    const wrapper = mount(LinkTab, { props: { path: 'links/clip.md', links: createLinks('', '') } })
    await flushPromises()
    expect(wrapper.text()).toContain('No text found at this link.')
  })

  it('says when the person deleted the text', async () => {
    const wrapper = mount(LinkTab, {
      props: { path: 'links/clip.md', links: createLinks('', '', true) },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('The text of this link was deleted.')
  })
})

describe('LinkTab without a copy', () => {
  it('frames the page that holds the video player', async () => {
    const wrapper = mount(LinkTab, {
      props: {
        path: 'links/clip.md',
        links: createLinks('', '', false, 'plugin://md.numen.tools.media.importer/clip.html'),
      },
    })
    await flushPromises()
    const frame = wrapper.find('iframe.link-tab__embed')
    expect(frame.attributes('src')).toBe('http://127.0.0.1:1/t/plugin/v/importer/clip.html?size=1')
    expect(wrapper.find('video').exists()).toBe(false)
  })

  it('frames nothing for an address with no player', async () => {
    const wrapper = mount(LinkTab, { props: { path: 'links/clip.md', links: createLinks('', '') } })
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
  })
})

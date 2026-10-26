import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import LinkTab from './LinkTab.vue'
import type { Links } from './types'

const createLinks = (copySize: number, text: string): Links => ({
  getLink: async () => ({
    ok: true,
    value: { url: 'https://videos.example/clip', isCopyable: true, copySize },
  }),
  readArticle: async () => text,
  getSource: async () => ({
    mediaUrl: 'http://127.0.0.1:1/t/v/links/clip.url',
    mediaType: 'audio/wav',
  }),
  createLink: async () => ({ ok: false, error: '' }),
  watchWork: () => () => {},
})

describe('LinkTab', () => {
  it('shows the text fetched from the link', async () => {
    const wrapper = mount(LinkTab, {
      props: {
        state: { path: 'links/rance.url' },
        links: createLinks(0, 'A tidal mill stored seawater.'),
      },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('A tidal mill stored seawater.')
    expect(wrapper.find('audio').exists()).toBe(false)
    expect(wrapper.find('a').attributes('href')).toBe('https://videos.example/clip')
  })

  it('plays the copy of the video from where the window serves it', async () => {
    const wrapper = mount(LinkTab, {
      props: { path: 'links/clip.url', links: createLinks(6000, '') },
    })
    await flushPromises()
    expect(wrapper.find('audio').attributes('src')).toBe('http://127.0.0.1:1/t/v/links/clip.url')
    expect(wrapper.get('button').text()).toBe('Play')
    expect(wrapper.text()).toContain('0:00')
  })

  it('says when nothing was fetched', async () => {
    const wrapper = mount(LinkTab, { props: { path: 'links/clip.url', links: createLinks(0, '') } })
    await flushPromises()
    expect(wrapper.text()).toContain('No text found at this link.')
  })
})

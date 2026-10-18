import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import MediaUrlTab from './MediaUrlTab.vue'
import { downloaderClient } from './client'
import { WORDS as words } from './words'

vi.mock('./client', () => ({
  downloaderClient: {
    canHandle: vi.fn(),
    getText: vi.fn(),
    downloadMedia: vi.fn(),
  },
}))

describe('MediaUrlTab', () => {
  it('renders input form initially', () => {
    const wrapper = mount(MediaUrlTab, { props: { state: {} } })
    expect(wrapper.find('input[type="url"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').text()).toBe(words.download)
  })

  it('names the address field by its label', () => {
    const wrapper = mount(MediaUrlTab, { props: { state: {} }, attachTo: document.body })
    const field = wrapper.get('input[type="url"]').element as HTMLInputElement

    expect([...(field.labels ?? [])].map((label) => label.textContent)).toEqual([words.mediaUrl])
    wrapper.unmount()
  })

  it('shows error when url cannot be handled', async () => {
    vi.mocked(downloaderClient.canHandle).mockResolvedValueOnce({ canHandle: false } as never)
    const wrapper = mount(MediaUrlTab, { props: { state: {} } })

    const input = wrapper.find('input[type="url"]')
    await input.setValue('https://example.com/unsupported')
    await wrapper.find('form').trigger('submit.prevent')

    expect(downloaderClient.canHandle).toHaveBeenCalledWith({
      url: 'https://example.com/unsupported',
    })
    expect(wrapper.text()).toContain(words.unsupportedUrl)
  })

  it('fetches media metadata and cues when url is handled', async () => {
    vi.mocked(downloaderClient.canHandle).mockResolvedValueOnce({ canHandle: true } as never)
    vi.mocked(downloaderClient.getText).mockResolvedValueOnce({
      title: 'Sample Video',
      cues: [{ startMs: 1000, endMs: 2000, text: 'First line' }],
    } as never)

    const wrapper = mount(MediaUrlTab, { props: { state: {} } })
    const input = wrapper.find('input[type="url"]')
    await input.setValue('https://example.com/video')
    await wrapper.find('form').trigger('submit.prevent')

    expect(downloaderClient.getText).toHaveBeenCalledWith({ url: 'https://example.com/video' })
    expect(wrapper.text()).toContain('Sample Video')
    expect(wrapper.text()).toContain('First line')
  })
})

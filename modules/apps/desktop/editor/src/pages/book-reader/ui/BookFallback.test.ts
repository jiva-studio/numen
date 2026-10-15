import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BookFallback from './BookFallback.vue'
import BookTab from './BookTab.vue'
import { isBookPluginActive, setBookPluginActive } from '../model/pluginState'
import { ref } from 'vue'

describe('BookFallback', () => {
  it('renders clean fallback screen with filename and plugin requirement notice', () => {
    const wrapper = mount(BookFallback, {
      props: {
        path: 'library/mahabharata.epub',
      },
    })

    expect(wrapper.text()).toContain('mahabharata.epub')
    expect(wrapper.text()).toContain('Plugin Required')
    expect(wrapper.text()).toContain('md.numen.epub')
  })

  it('renders fallback in BookTab when plugin is disabled', async () => {
    setBookPluginActive(false)
    expect(isBookPluginActive.value).toBe(false)

    const fakeState = {
      path: 'library/sample.epub',
      title: ref('Sample Book'),
      isLoading: ref(false),
      contents: ref([]),
      offset: ref(0),
      markup: ref(''),
      drawn: ref(''),
      reading: ref({ from: 0, to: 0 }),
      span: ref({ from: 0, to: 0 }),
      highlights: ref([]),
      otherHighlights: ref([]),
      chapter: ref(''),
      setBookHandle: () => {},
      setTabElement: () => {},
      handleKeyPress: () => false,
      goToOffset: async () => {},
      followLink: async () => {},
    } as any

    const wrapper = mount(BookTab, {
      props: {
        state: fakeState,
      },
    })

    expect(wrapper.findComponent(BookFallback).exists()).toBe(true)
    expect(wrapper.text()).toContain('sample.epub')
    expect(wrapper.text()).toContain('md.numen.epub')

    // Restore plugin active state
    setBookPluginActive(true)
  })
})

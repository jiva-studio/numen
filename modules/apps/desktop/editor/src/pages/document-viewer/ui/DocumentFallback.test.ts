import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DocumentFallback from './DocumentFallback.vue'
import DocumentTab from './DocumentTab.vue'
import { isPdfPluginActive, setPdfPluginActive } from '../model/pluginState'
import { ref } from 'vue'

describe('DocumentFallback', () => {
  it('renders clean fallback screen with filename and plugin requirement notice', () => {
    const wrapper = mount(DocumentFallback, {
      props: {
        path: 'library/sample.pdf',
      },
    })

    expect(wrapper.text()).toContain('sample.pdf')
    expect(wrapper.text()).toContain('Plugin Required')
    expect(wrapper.text()).toContain('md.numen.pdf')
  })

  it('renders fallback in DocumentTab when plugin is disabled', async () => {
    setPdfPluginActive(false)
    expect(isPdfPluginActive.value).toBe(false)

    const fakeState = {
      path: 'library/sample.pdf',
      pages: ref([]),
      pageNumber: ref(0),
      isLoading: ref(false),
      error: ref(''),
      getPageImageUrl: () => '',
      getHighlightsOn: () => [],
      getOtherHighlightsOn: () => [],
      goToPage: async () => {},
      widen: () => {},
      setPageHandle: () => {},
      measure: () => {},
      handleKeyPress: () => false,
      focusTab: () => {},
      close: () => {},
    } as any

    const wrapper = mount(DocumentTab, {
      props: {
        state: fakeState,
      },
    })

    expect(wrapper.findComponent(DocumentFallback).exists()).toBe(true)
    expect(wrapper.text()).toContain('sample.pdf')
    expect(wrapper.text()).toContain('md.numen.pdf')

    // Restore plugin active state
    setPdfPluginActive(true)
  })
})

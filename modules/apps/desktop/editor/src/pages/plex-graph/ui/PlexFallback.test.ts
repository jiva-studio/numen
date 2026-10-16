import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PlexFallback from './PlexFallback.vue'
import PlexTab from './PlexTab.vue'
import { isPlexPluginActive, setPlexPluginActive } from '../model/pluginState'

describe('PlexFallback', () => {
  it('renders clean fallback screen with plugin requirement notice', () => {
    const wrapper = mount(PlexFallback)

    expect(wrapper.text()).toContain('Knowledge Graph Visualizer')
    expect(wrapper.text()).toContain('Plugin Required')
    expect(wrapper.text()).toContain('md.numen.view-plex-graph')
  })

  it('renders fallback in PlexTab when plugin is disabled', async () => {
    setPlexPluginActive(false)
    expect(isPlexPluginActive.value).toBe(false)

    const fakeState = {
      view: { error: { value: '' }, isLoading: { value: false } },
    } as any

    const wrapper = mount(PlexTab, {
      props: {
        state: fakeState,
      },
    })

    expect(wrapper.findComponent(PlexFallback).exists()).toBe(true)
    expect(wrapper.text()).toContain('Knowledge Graph Visualizer')
    expect(wrapper.text()).toContain('md.numen.view-plex-graph')

    // Restore plugin active state
    setPlexPluginActive(true)
  })
})

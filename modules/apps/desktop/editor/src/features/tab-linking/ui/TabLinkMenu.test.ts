import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TabLinkMenu from './TabLinkMenu.vue'
import { UNLINK_ID } from '../lib/menu'

describe('TabLinkMenu', () => {
  const eligibleTabs = [
    { id: 'note:alpha', title: 'Alpha Note', kind: 'note' as const },
    { id: 'plex:graph', title: 'Graph View', kind: 'plex' as const },
  ]

  it('renders menu and emits link when tab item chosen', async () => {
    const wrapper = mount(TabLinkMenu, {
      props: {
        tabId: 'note:main',
        eligibleTabs,
        at: { x: 100, y: 100 },
        isLinked: false,
      },
    })

    const menu = wrapper.findComponent({ name: 'Menu' })
    expect(menu.exists()).toBe(true)

    menu.vm.$emit('choose', 'note:alpha')
    expect(wrapper.emitted('link')?.[0]).toEqual(['note:alpha'])
  })

  it('emits unlink when unlink item chosen', async () => {
    const wrapper = mount(TabLinkMenu, {
      props: {
        tabId: 'note:main',
        eligibleTabs,
        at: { x: 100, y: 100 },
        isLinked: true,
        linkedTargetTitle: 'Alpha Note',
      },
    })

    const menu = wrapper.findComponent({ name: 'Menu' })
    menu.vm.$emit('choose', UNLINK_ID)
    expect(wrapper.emitted('unlink')).toBeTruthy()
  })

  it('emits close when menu is dismissed', async () => {
    const wrapper = mount(TabLinkMenu, {
      props: {
        tabId: 'note:main',
        eligibleTabs,
        at: { x: 100, y: 100 },
      },
    })

    const menu = wrapper.findComponent({ name: 'Menu' })
    menu.vm.$emit('dismiss')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})

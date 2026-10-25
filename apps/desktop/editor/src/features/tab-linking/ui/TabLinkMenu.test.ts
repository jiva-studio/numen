import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import TabLinkMenu from './TabLinkMenu.vue'
import { UNLINK_ID } from '../lib/menu'
import type { EligibleLinkTab } from '../types'

const PlexIcon = defineComponent({ name: 'PlexIcon', render: () => h('svg') })
const getTabIcon = (tab: string) => (tab === 'plex:graph' ? PlexIcon : null)

describe('TabLinkMenu', () => {
  const eligibleTabs: readonly EligibleLinkTab[] = [
    { id: 'note:alpha', title: 'Alpha Note', kind: 'note', isLinked: false },
    { id: 'plex:graph', title: 'Graph View', kind: 'plex', isLinked: false },
  ]

  it('renders menu and emits link when tab item chosen', async () => {
    const wrapper = mount(TabLinkMenu, {
      props: {
        tabId: 'note:main',
        eligibleTabs,
        at: { x: 100, y: 100 },
        getTabIcon,
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
        getTabIcon,
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
        getTabIcon,
      },
    })

    const menu = wrapper.findComponent({ name: 'Menu' })
    menu.vm.$emit('dismiss')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})

describe('the icon a tab carries in the link menu', () => {
  const eligibleTabs: readonly EligibleLinkTab[] = [
    { id: 'note:alpha', title: 'Alpha Note', kind: 'note', isLinked: false },
    { id: 'plex:graph', title: 'Graph View', kind: 'plex', isLinked: false },
  ]

  it('is the one the window draws on the tab, and nothing where it draws none', () => {
    const wrapper = mount(TabLinkMenu, {
      props: { tabId: 'note:main', eligibleTabs, at: { x: 0, y: 0 }, getTabIcon },
    })

    expect(wrapper.findAllComponents(PlexIcon)).toHaveLength(1)
  })
})

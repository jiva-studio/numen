import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import TabLinkMenu from './TabLinkMenu.vue'
import { DETACH_ID, UNLINK_ID } from '../lib/menu'
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

  it('emits detach when detach item chosen', async () => {
    const wrapper = mount(TabLinkMenu, {
      props: {
        tabId: 'note:main',
        eligibleTabs,
        at: { x: 100, y: 100 },
        getTabIcon,
        canDetach: true,
      },
    })

    const menu = wrapper.findComponent({ name: 'Menu' })
    menu.vm.$emit('choose', DETACH_ID)
    expect(wrapper.emitted('detach')).toBeTruthy()
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

describe('TabLinkMenu icon of a tab of another window', () => {
  it('uses the icon of its kind', () => {
    const FilesIcon = defineComponent({ name: 'FilesIcon', render: () => h('svg') })
    const wrapper = mount(TabLinkMenu, {
      props: {
        tabId: 'plex:main',
        eligibleTabs: [{ id: 'files:other', title: 'Files', kind: 'files', isLinked: false }],
        at: { x: 0, y: 0 },
        getTabIcon: () => null,
        getKindIcon: (kind: string) => (kind === 'files' ? FilesIcon : null),
      },
    })

    expect(wrapper.findAllComponents(FilesIcon)).toHaveLength(1)
  })
})

describe('the Move to window items of the menu', () => {
  const WINDOWS = [
    { id: 'window-a', isMain: true, activeTitle: 'Orbits', activeKind: 'plex', tabCount: 2 },
  ]
  const PlexKindIcon = defineComponent({ name: 'PlexKindIcon', render: () => h('svg') })
  const mountMenu = () =>
    mount(TabLinkMenu, {
      props: {
        tabId: 'note:main',
        eligibleTabs: [],
        at: { x: 0, y: 0 },
        getTabIcon: () => null,
        getKindIcon: (kind: string) => (kind === 'plex' ? PlexKindIcon : null),
        windows: WINDOWS,
      },
    })

  it('emits move with the window id when a window item is chosen', () => {
    const wrapper = mountMenu()

    wrapper.findComponent({ name: 'Menu' }).vm.$emit('choose', '__move__:window-a')

    expect(wrapper.emitted('move')).toEqual([['window-a']])
    expect(wrapper.emitted('link')).toBeUndefined()
  })

  it('emits highlight with the window id under the pointer and null for other items', () => {
    const wrapper = mountMenu()
    const menu = wrapper.findComponent({ name: 'Menu' })

    menu.vm.$emit('highlight', '__move__:window-a')
    menu.vm.$emit('highlight', '__move__')
    menu.vm.$emit('highlight', null)

    expect(wrapper.emitted('highlight')).toEqual([['window-a'], [null], [null]])
  })

  it('shows the kind icon of the active tab of a window', () => {
    const MenuStub = defineComponent({
      name: 'MenuStub',
      props: { items: { type: Array, required: true } },
      setup(stubProps, { slots }) {
        const getIds = (items: readonly { id: string; children?: unknown[] }[]): string[] =>
          items.flatMap((item) => [
            item.id,
            ...getIds((item.children ?? []) as { id: string; children?: unknown[] }[]),
          ])
        return () =>
          h(
            'div',
            getIds(stubProps.items as { id: string }[]).map((id) => slots.icon?.({ id })),
          )
      },
    })
    const wrapper = mount(TabLinkMenu, {
      props: {
        tabId: 'note:main',
        eligibleTabs: [],
        at: { x: 0, y: 0 },
        getTabIcon: () => null,
        getKindIcon: (kind: string) => (kind === 'plex' ? PlexKindIcon : null),
        windows: WINDOWS,
      },
      global: { stubs: { Menu: MenuStub } },
    })

    expect(wrapper.findAllComponents(PlexKindIcon)).toHaveLength(1)
  })
})

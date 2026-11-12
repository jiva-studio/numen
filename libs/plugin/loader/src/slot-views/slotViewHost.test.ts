/** The view of a slot of the window, drawn with the stylesheet of its plugin. Every plugin is invented. */
import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref } from 'vue'
import { registerPluginStyle, unregisterPluginStyle } from '../styles'
import { SlotViewHost } from './slotViewHost'
import type { DeclaredSlotView } from './slotViews'

const PLUGIN = 'org.example.bar'

const createView = (pluginId = PLUGIN): DeclaredSlotView => ({
  pluginId,
  slot: 'statusBar',
  registration: {
    id: 'bar',
    component: defineComponent({ setup: () => () => h('div', { 'data-bar': pluginId }) }),
  },
})

const getLinks = () => [...document.head.querySelectorAll<HTMLLinkElement>('link[data-plugin-id]')]

afterEach(() => {
  unregisterPluginStyle(PLUGIN)
  document.head.innerHTML = ''
})

describe('the host of a slot view', () => {
  it('draws nothing without a view', () => {
    expect(mount(SlotViewHost, { props: { view: null } }).element.nodeType).toBe(Node.COMMENT_NODE)
  })

  it('draws the view with the stylesheet of its plugin, and removes the stylesheet when the view is removed', async () => {
    registerPluginStyle(PLUGIN, 'http://127.0.0.1:1/plugins/bar/style.css')
    const view = ref<DeclaredSlotView | null>(createView())
    const wrapper = mount(
      defineComponent({ setup: () => () => h(SlotViewHost, { view: view.value }) }),
    )

    expect(wrapper.find(`[data-bar="${PLUGIN}"]`).exists()).toBe(true)
    expect(getLinks().map((one) => one.dataset['pluginId'])).toStrictEqual([PLUGIN])

    view.value = null
    await nextTick()

    expect(getLinks()).toStrictEqual([])
  })
})

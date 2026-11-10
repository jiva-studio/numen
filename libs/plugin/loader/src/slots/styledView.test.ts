/** The views of a slot drawn by another plugin carry the stylesheet of their own plugin. Every plugin here is invented. */
import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, type Component } from 'vue'
import { clearPluginStyles, registerPluginStyle } from '../styles'
import { createSlotViewRegistry } from '../slot-views'

const LEDGER = 'org.example.ledger'
const BALANCE = defineComponent({
  props: { pluginId: { type: String, default: '' } },
  setup: (props) => () => h('em', { class: 'balance' }, `Balance of ${props.pluginId}`),
})

afterEach(() => {
  clearPluginStyles()
  document.head.innerHTML = ''
})

describe('a view a slot serves', () => {
  it('is drawn with the stylesheet of the plugin that registered it, until it is taken away', () => {
    registerPluginStyle(LEDGER, '/plugins/org.example.ledger/dist/styles.css')
    const registry = createSlotViewRegistry()
    registry.registerSlotView(LEDGER, 'titleBar.status', { id: 'balance', component: BALANCE })

    const wrapper = mount(registry.findView(LEDGER, 'balance') as Component, {
      props: { pluginId: LEDGER },
      attachTo: document.body,
    })

    const link = wrapper.find(`link[data-plugin-id="${LEDGER}"]`)
    expect(link.attributes('href')).toBe('/plugins/org.example.ledger/dist/styles.css')
    expect(wrapper.find('em.balance').text()).toBe(`Balance of ${LEDGER}`)
    wrapper.unmount()
    expect(document.querySelector(`link[data-plugin-id="${LEDGER}"]`)).toBeNull()
  })

  it('is the same component on every read, so that its owner does not draw it again', () => {
    const registry = createSlotViewRegistry()
    registry.registerSlotView(LEDGER, 'titleBar.status', { id: 'balance', component: BALANCE })

    expect(registry.findView(LEDGER, 'balance')).toBe(registry.findView(LEDGER, 'balance'))
  })

  it('draws without a stylesheet where its plugin has none', () => {
    const registry = createSlotViewRegistry()
    registry.registerSlotView(LEDGER, 'titleBar.status', { id: 'balance', component: BALANCE })

    const wrapper = mount(registry.findView(LEDGER, 'balance') as Component)

    expect(wrapper.find('link').exists()).toBe(false)
    expect(wrapper.find('em.balance').exists()).toBe(true)
  })
})

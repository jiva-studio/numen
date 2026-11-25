/** A plugin dialog that throws is reported for closing, and the dialogs beside it stay drawn. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import type { OpenDialog } from '../model/createDialogStack'
import PluginDialogs from './PluginDialogs.vue'

const fine = defineComponent({ render: () => h('p', 'fine') })
const broken = defineComponent({
  setup: () => {
    throw new Error('the plugin dialog is broken')
  },
})

describe('PluginDialogs', () => {
  it('draws each open dialog', () => {
    const entries: OpenDialog[] = [{ id: 1, options: { component: fine } }]
    const wrapper = mount(PluginDialogs, { props: { entries } })
    expect(wrapper.text()).toBe('fine')
  })

  it('reports the dialog that throws while it draws and keeps the others', () => {
    const entries: OpenDialog[] = [
      { id: 1, options: { component: fine } },
      { id: 2, options: { component: broken } },
    ]
    const wrapper = mount(PluginDialogs, { props: { entries } })
    expect(wrapper.emitted('fail')).toStrictEqual([[2]])
    expect(wrapper.text()).toBe('fine')
  })
})

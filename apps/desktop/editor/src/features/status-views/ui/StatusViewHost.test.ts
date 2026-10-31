import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import StatusViewHost from './StatusViewHost.vue'

const widget = (text: string) =>
  defineComponent({
    props: { pluginId: { type: String, required: true } },
    setup: (props) => () => h('span', { class: 'widget' }, `${text}:${props.pluginId}`),
  })

describe('StatusViewHost', () => {
  it('draws nothing while no plugin registered a view', () => {
    const wrapper = mount(StatusViewHost, { props: { views: [] } })

    expect(wrapper.findAll('.widget')).toHaveLength(0)
  })

  it('draws each registered component in order, passing the id of its plugin', () => {
    const wrapper = mount(StatusViewHost, {
      props: {
        views: [
          { pluginId: 'a.one', registration: { id: 'first', component: widget('first') } },
          { pluginId: 'b.two', registration: { id: 'first', component: widget('second') } },
        ],
      },
    })

    expect(wrapper.findAll('.widget').map((one) => one.text())).toEqual([
      'first:a.one',
      'second:b.two',
    ])
  })
})

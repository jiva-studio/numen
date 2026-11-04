import { afterEach, describe, expect, it, vi } from 'vitest'
import * as pluginHost from '@numen/plugin-host'
import { clearPluginStyles, registerPluginStyle } from '@numen/plugin-host'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import StatusViewHost from './StatusViewHost.vue'

const widget = (text: string) =>
  defineComponent({
    props: { pluginId: { type: String, required: true } },
    setup: (props) => () => h('span', { class: 'widget' }, `${text}:${props.pluginId}`),
  })

describe('StatusViewHost', () => {
  afterEach(() => {
    clearPluginStyles()
    document.head.querySelectorAll('link[data-plugin-id]').forEach((one) => one.remove())
  })

  it('draws nothing while no plugin registered a view', () => {
    const wrapper = mount(StatusViewHost, { props: { views: [] } })

    expect(wrapper.findAll('.widget')).toHaveLength(0)
    expect(document.head.querySelectorAll('link[data-plugin-id]')).toHaveLength(0)
    wrapper.unmount()
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
    expect(document.head.querySelectorAll('link[data-plugin-id]')).toHaveLength(0)
    wrapper.unmount()
  })

  it('adds the stylesheet of a plugin with a view to the window, and takes it out with the view', async () => {
    registerPluginStyle('a.one', '/plugins/a.one/dist/styles.css')
    registerPluginStyle('b.two', '/plugins/b.two/dist/styles.css')
    const wrapper = mount(StatusViewHost, {
      props: {
        views: [{ pluginId: 'a.one', registration: { id: 'first', component: widget('first') } }],
      },
    })
    const linked = () =>
      [...document.head.querySelectorAll<HTMLLinkElement>('link[data-plugin-id]')].map(
        (one) => one.dataset['pluginId'],
      )

    const firstLink = document.head.querySelector<HTMLLinkElement>('link[data-plugin-id="a.one"]')
    expect(firstLink?.rel).toBe('stylesheet')
    expect(firstLink?.getAttribute('href')).toBe('/plugins/a.one/dist/styles.css')
    expect(linked()).toEqual(['a.one'])

    await wrapper.setProps({
      views: [
        { pluginId: 'a.one', registration: { id: 'first', component: widget('first') } },
        { pluginId: 'b.two', registration: { id: 'second', component: widget('second') } },
      ],
    })
    expect(linked()).toEqual(['a.one', 'b.two'])
    expect(document.head.querySelector('link[data-plugin-id="a.one"]')).toBe(firstLink)
    expect(document.head.querySelectorAll('link[data-plugin-id="a.one"]')).toHaveLength(1)

    await wrapper.setProps({ views: [] })
    expect(linked()).toEqual([])

    await wrapper.setProps({
      views: [{ pluginId: 'a.one', registration: { id: 'first', component: widget('first') } }],
    })
    expect(linked()).toEqual(['a.one'])

    wrapper.unmount()
    expect(linked()).toEqual([])
  })

  it('adopts an inline stylesheet when css text is registered and releases it on unmount', () => {
    const release = vi.fn()
    const spy = vi.spyOn(pluginHost, 'adoptPluginStyle').mockReturnValueOnce(release)
    try {
      registerPluginStyle('a.one', '/plugins/a.one/dist/styles.css')
      const wrapper = mount(StatusViewHost, {
        props: {
          views: [{ pluginId: 'a.one', registration: { id: 'first', component: widget('first') } }],
        },
      })

      expect(document.head.querySelectorAll('link[data-plugin-id]')).toHaveLength(0)
      expect(spy).toHaveBeenCalledWith('a.one', document)

      wrapper.unmount()
      expect(release).toHaveBeenCalledOnce()
      expect(spy).toHaveBeenCalledOnce()
    } finally {
      spy.mockRestore()
    }
  })
})

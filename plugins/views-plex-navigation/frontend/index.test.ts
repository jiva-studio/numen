/** The view of the plugin as the board of the plex draws it. */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from './index'

const listDisabledOf = (wrapper: VueWrapper) =>
  wrapper.findAll('button').map((one) => one.element.disabled)

const setup = (board = 'plex:1', path = 'one.md') => {
  const plugin = createMockPluginContext()
  activate(plugin, createMockHostServices())
  const component = plugin.getRegisteredSlotViews()[0]?.component as Component
  const go = vi.fn()
  const wrapper = mount(component, { props: { board, path, go } })
  const moveTo = async (next: string) => {
    await wrapper.setProps({ path: next })
    await flushPromises()
  }
  return { plugin, wrapper, go, moveTo }
}

describe('the view of the plugin', () => {
  it('registers the view the manifest places at the top of the board', () => {
    const { plugin } = setup()

    expect(plugin.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual(['navigation'])
  })

  it('disables both buttons on the first node', () => {
    expect(listDisabledOf(setup().wrapper)).toStrictEqual([true, true])
  })

  it('enables Back after the board moves to another node', async () => {
    const { wrapper, moveTo } = setup()
    await moveTo('two.md')

    expect(listDisabledOf(wrapper)).toStrictEqual([false, true])
  })

  it('calls go with the previous node on Back and enables Forward', async () => {
    const { wrapper, go, moveTo } = setup()
    await moveTo('two.md')
    await wrapper.findAll('button')[0]?.trigger('click')
    await moveTo('one.md')

    expect(go).toHaveBeenCalledWith('one.md')
    expect(listDisabledOf(wrapper)).toStrictEqual([true, false])
  })

  it('calls go with the next node on Forward', async () => {
    const { wrapper, go, moveTo } = setup()
    await moveTo('two.md')
    await wrapper.findAll('button')[0]?.trigger('click')
    await moveTo('one.md')
    await wrapper.findAll('button')[1]?.trigger('click')

    expect(go).toHaveBeenLastCalledWith('two.md')
  })

  it('drops the forward nodes when the board moves to a new node after Back', async () => {
    const { wrapper, moveTo } = setup()
    await moveTo('two.md')
    await wrapper.findAll('button')[0]?.trigger('click')
    await moveTo('one.md')
    await moveTo('three.md')

    expect(listDisabledOf(wrapper)).toStrictEqual([false, true])
  })
})

import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, SUPPORTER_VIEW } from './index'
import { SUPPORTER_LICENSE_ID } from './model/supporter'

const setup = () => {
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.supporter' })
  activate(context)
  const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)
  return { context, wrapper }
}

describe('the supporter plugin frontend', () => {
  it('registers the supporter status view', () => {
    const { context } = setup()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual([SUPPORTER_VIEW])
  })

  it('draws the badge while the supporter license is active and hides it when deactivated', async () => {
    const { context, wrapper } = setup()

    expect(wrapper.find('.supporter-badge').exists()).toBe(false)
    context.setLicense(SUPPORTER_LICENSE_ID, true)
    await nextTick()
    expect(wrapper.find('.supporter-badge').exists()).toBe(true)

    context.setLicense(SUPPORTER_LICENSE_ID, false)
    await nextTick()
    expect(wrapper.find('.supporter-badge').exists()).toBe(false)
  })

  it('stops listening when the plugin context is disposed', async () => {
    const { context, wrapper } = setup()

    context.dispose()
    context.setLicense(SUPPORTER_LICENSE_ID, true)
    await nextTick()

    expect(wrapper.find('.supporter-badge').exists()).toBe(false)
  })
})

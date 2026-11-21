/** The view of the plugin as the board of the plex draws it. */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, createNavigationView } from './index'
import { createBoardHistories } from './model/history'

describe('the view of the plugin', () => {
  it('registers the view the manifest places at the top of the board', () => {
    const plugin = createMockPluginContext()
    activate(plugin, createMockHostServices())

    expect(plugin.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual(['navigation'])
  })

  it('enables the buttons the history of its board allows', () => {
    const histories = createBoardHistories()
    histories.visit('a', 'one.md')
    histories.visit('a', 'two.md')
    const wrapper = mount(createNavigationView({ histories, getBoard: () => 'a' }))

    expect(wrapper.findAll('button').map((one) => one.element.disabled)).toStrictEqual([
      false,
      true,
    ])
  })

  it('disables both buttons for a board that is not known', () => {
    const wrapper = mount(
      createNavigationView({ histories: createBoardHistories(), getBoard: () => null }),
    )

    expect(wrapper.findAll('button').every((one) => one.element.disabled)).toBe(true)
  })
})

import { mount } from '@vue/test-utils'
import { defineComponent, h, shallowRef } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { createWindowTabs } from '@/entities/tab'
import { useWindowKeys } from './useWindowKeys'

const VAULTS = [{ id: 'physics', name: 'Physics', path: '/vaults/Physics' }]

const setup = (isRegistered: boolean) => {
  const tabs = createWindowTabs()
  const openCalls = vi.fn()
  if (isRegistered) {
    tabs.registerKind({
      kind: 'review',
      pluginId: 'md.numen.cards',
      pane: {},
      display: { placement: { pane: 'main', order: 0 } },
      open: (vault) => {
        openCalls(vault)
        return {}
      },
    })
  }
  const wrapper = mount(
    defineComponent({
      setup() {
        useWindowKeys({ tabs, vaults: shallowRef(VAULTS) })
        return () => h('div')
      },
    }),
  )
  return { tabs, openCalls, wrapper }
}

const press = (key: string) =>
  window.dispatchEvent(new KeyboardEvent('keydown', { key, cancelable: true }))

describe('useWindowKeys', () => {
  it('opens the tab on the vault at the letter while mounted', () => {
    const { tabs, openCalls, wrapper } = setup(true)

    press('a')
    expect(openCalls).toHaveBeenCalledExactlyOnceWith('physics')
    expect(tabs.tabVault.value).toBe('physics')

    tabs.closeTab()
    wrapper.unmount()
    press('a')
    expect(openCalls).toHaveBeenCalledTimes(1)
    expect(tabs.tab.value).toBeNull()
  })

  it('opens nothing while no plugin registered a front kind', () => {
    const { tabs, openCalls } = setup(false)

    press('a')

    expect(openCalls).not.toHaveBeenCalled()
    expect(tabs.tab.value).toBeNull()
  })
})

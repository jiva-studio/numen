/**
 * Where the keyboard focus goes when the confirmation of a deletion closes. The secrets here are
 * invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import SettingsTab from '../SettingsTab.vue'
import { createInstallation } from '@/testing/installation'
import { click, dispose, findButton, root, rowOf } from '@/testing/secretsDom'

const secrets = [
  { id: 's00000001', name: 'Work', group: 'harbour', allowed: [], hasValue: true },
  { id: 's00000002', name: 'Home', group: 'harbour', allowed: [], hasValue: true },
]

const mountPage = () => {
  const installation = createInstallation({ secrets: ref(secrets) })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: { installation, initialPluginId: ref('secrets') } },
  })
  return { installation, wrapper }
}

const openDialog = async (name: string): Promise<{ opener: HTMLElement; dialog: HTMLElement }> => {
  const opener = findButton(/delete secret/i, rowOf(name))
  opener.focus()
  await click(opener)
  return { opener, dialog: root().querySelector('[role="alertdialog"]') as HTMLElement }
}

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

describe('the confirmation of a deletion', () => {
  it('returns the focus to the Delete button that opened it after Cancel', async () => {
    const { wrapper } = mountPage()
    const { opener, dialog } = await openDialog('Home')
    await click(findButton(/cancel/i, dialog))
    expect(document.activeElement).toBe(opener)
    dispose(wrapper)
  })

  it('returns the focus to the Delete button that opened it after Confirm while the button is still there', async () => {
    const { wrapper } = mountPage()
    const { opener, dialog } = await openDialog('Home')
    await click(findButton(/^delete$/i, dialog))
    expect(document.activeElement).toBe(opener)
    dispose(wrapper)
  })
})

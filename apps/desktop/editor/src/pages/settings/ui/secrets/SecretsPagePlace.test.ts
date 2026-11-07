/**
 * What the page of secrets says about where the values are kept. The secrets here are invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import SettingsTab from '../SettingsTab.vue'
import { createInstallation } from '@/testing/installation'
import { dispose, root } from '@/testing/secretsDom'

const mountPage = (overrides: Parameters<typeof createInstallation>[0]) => {
  const installation = createInstallation(overrides)
  return mount(SettingsTab, {
    attachTo: document.body,
    props: { state: { installation, initialPluginId: ref('secrets') } },
  })
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

describe('the place of the values on the page of secrets', () => {
  it('names the keychain and the fallback file where no file is used', () => {
    const wrapper = mountPage({ credentialsPath: ref('') })
    expect(root().textContent).toMatch(
      /kept in the system keychain or, where it is not available, in the file credentials\/credentials\.json/i,
    )
    dispose(wrapper)
  })

  it('names the file where the key store keeps its values in a file', () => {
    const wrapper = mountPage({ credentialsPath: ref('/home/someone/credentials.json') })
    expect(root().textContent).toContain('/home/someone/credentials.json')
    dispose(wrapper)
  })
})

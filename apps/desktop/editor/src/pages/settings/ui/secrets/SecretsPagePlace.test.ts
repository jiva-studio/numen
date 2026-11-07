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
  it('names the keychain where a key store exists and no file is used', () => {
    const wrapper = mountPage({ hasKeyStore: ref(true), credentialsPath: ref('') })
    expect(root().textContent).toMatch(/system keychain/i)
    dispose(wrapper)
  })

  it('names the file where the key store keeps its values in a file', () => {
    const wrapper = mountPage({
      hasKeyStore: ref(true),
      credentialsPath: ref('/home/someone/credentials.json'),
    })
    expect(root().textContent).toContain('/home/someone/credentials.json')
    dispose(wrapper)
  })

  it('says nothing about the place where no key store exists', () => {
    const wrapper = mountPage({ hasKeyStore: ref(false), credentialsPath: ref('') })
    expect(root().textContent).not.toMatch(/keychain/i)
    expect(root().querySelector('.secrets-page__where')).toBeNull()
    dispose(wrapper)
  })
})

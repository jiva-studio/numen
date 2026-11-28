/**
 * The page "Identity and licenses" draws the views plugins registered for it between the identity and the
 * license cards. Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { LicenseInfo } from '@/entities/license'
import { createInstallation, createSettingsState } from '@/pages/settings/testing/installation'
import SettingsTab from '../SettingsTab.vue'
import { WORDS as words } from '../../words'

const SUBSCRIPTION_CARD = defineComponent({
  name: 'SubscriptionCard',
  props: { pluginId: { type: String, default: '' } },
  setup: (props) => () =>
    h('section', { 'data-testid': 'plugin-view' }, `Card of ${props.pluginId}`),
})

const SYNC: LicenseInfo = {
  id: 'numen.sync',
  title: 'Invented license',
  contributors: [],
  providers: [],
  status: 'inactive',
  isActive: false,
  hasKey: false,
  keyHint: '',
  activeProvider: '',
  instanceName: '',
  expiresUnixMs: 0,
  verifiedUnixMs: 0,
}

const mounted: VueWrapper[] = []

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
  while (mounted.length) mounted.pop()?.unmount()
})

const openPage = async (options: {
  readonly views: readonly { pluginId: string; id: string }[]
  readonly licenses: readonly LicenseInfo[]
}) => {
  const installation = createInstallation({
    getIdentityViews: () => options.views.map((one) => ({ ...one, component: SUBSCRIPTION_CARD })),
    licenses: {
      list: ref([...options.licenses]),
      load: vi.fn(() => Promise.resolve()),
      watch: vi.fn(() => Promise.resolve()),
      activate: vi.fn(),
      deactivate: vi.fn(),
    },
  })
  const tab = mount(SettingsTab, {
    props: { state: createSettingsState(installation, { initialSearchQuery: ref('') }) },
    attachTo: document.body,
  })
  mounted.push(tab)
  await tab.get('button[aria-label="Identity and licenses"]').trigger('click')
  await flushPromises()
  return tab
}

describe('the views of plugins on the page "Identity and licenses"', () => {
  it('draws the view of a plugin and gives it the id of the plugin', async () => {
    const tab = await openPage({
      views: [{ pluginId: 'md.numen.tools.sync', id: 'subscription' }],
      licenses: [],
    })

    expect(tab.get('[data-testid="plugin-view"]').text()).toBe('Card of md.numen.tools.sync')
  })

  it('draws the views in the order the installation lists them', async () => {
    const tab = await openPage({
      views: [
        { pluginId: 'a.first', id: 'one' },
        { pluginId: 'b.second', id: 'two' },
      ],
      licenses: [],
    })

    expect(tab.findAll('[data-testid="plugin-view"]').map((one) => one.text())).toStrictEqual([
      'Card of a.first',
      'Card of b.second',
    ])
  })

  it('draws the view after the identity and before the license cards', async () => {
    const tab = await openPage({ views: [{ pluginId: 'a.first', id: 'one' }], licenses: [SYNC] })

    const text = tab.text()
    expect(text.indexOf('No identity on this device')).toBeLessThan(text.indexOf('Card of a.first'))
    expect(text.indexOf('Card of a.first')).toBeLessThan(text.indexOf('Invented license'))
  })

  it('does not say that no license exists when a plugin view is drawn', async () => {
    const tab = await openPage({ views: [{ pluginId: 'a.first', id: 'one' }], licenses: [] })

    expect(tab.text()).not.toContain(words.licensesEmpty)
  })

  it('says that no license exists when there is no license and no plugin view', async () => {
    const tab = await openPage({ views: [], licenses: [] })

    expect(tab.text()).toContain(words.licensesEmpty)
  })
})

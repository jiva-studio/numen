import { describe, expect, it, vi } from 'vitest'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from './index'
import VaultsView from './ui/VaultsView.vue'

describe('the settings views of the sync plugin', () => {
  it('registers the vaults, shares and subscription views the manifest declares', () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(Response.json({ hasIdentity: false, isReachable: true, items: [] })),
    )
    const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })

    activate(context, createMockHostServices())

    const views = context.getRegisteredSettingsViews()
    expect(views.map((one) => one.id)).toStrictEqual(['vaults', 'shares', 'subscription'])
    expect(views.at(0)).toMatchObject({ component: VaultsView })
    expect(views.at(1)).toMatchObject({ component: { name: 'SharesPage' } })
    context.dispose()
    vi.unstubAllGlobals()
  })
})

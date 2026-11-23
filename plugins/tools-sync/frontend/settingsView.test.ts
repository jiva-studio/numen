import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from './index'
import VaultsView from './ui/VaultsView.vue'

describe('the settings views of the sync plugin', () => {
  it('registers the vaults view and the subscription view the manifest declares', () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(Response.json({ hasIdentity: false, isReachable: true, items: [] })),
    )
    const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })

    activate(context)

    const views = context.getRegisteredSettingsViews()
    expect(views.map((one) => one.id)).toStrictEqual(['vaults', 'subscription'])
    expect(views.at(0)).toMatchObject({ component: VaultsView })
    context.dispose()
    vi.unstubAllGlobals()
  })
})

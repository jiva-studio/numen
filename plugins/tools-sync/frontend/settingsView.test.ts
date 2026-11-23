import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from './index'
import VaultsView from './ui/VaultsView.vue'

describe('the settings view of the sync plugin', () => {
  it('registers the vaults view the manifest declares', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })

    activate(context)

    const views = context.getRegisteredSettingsViews()
    expect(views).toHaveLength(1)
    expect(views.at(0)).toMatchObject({ id: 'vaults', component: VaultsView })
  })
})

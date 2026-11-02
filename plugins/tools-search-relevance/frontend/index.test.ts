import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { describe, expect, it } from 'vitest'
import FloorPanel from './floor/ui/FloorPanel.vue'
import { activate } from './index'

describe('activate', () => {
  it('registers the floor settings view', () => {
    const context = createMockPluginContext()
    activate(context)
    const views = context.getRegisteredSettingsViews()
    expect(views).toHaveLength(1)
    expect(views[0]).toMatchObject({ id: 'floor', component: FloorPanel })
  })
})

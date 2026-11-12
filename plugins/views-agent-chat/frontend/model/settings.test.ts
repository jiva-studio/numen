import { describe, expect, it } from 'vitest'
import { SETTINGS_OPEN_COMMAND } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { createSettingsOpener } from './settings'

describe('the settings of the agent chat', () => {
  it('open on the page of the plugin through the command of the window', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.agent-chat' })

    createSettingsOpener(context)()

    expect(context.ui.commands.getRuns()).toStrictEqual([
      {
        command: SETTINGS_OPEN_COMMAND,
        target: undefined,
        itemId: undefined,
        values: { plugin: 'md.numen.views.agent-chat' },
      },
    ])
  })
})

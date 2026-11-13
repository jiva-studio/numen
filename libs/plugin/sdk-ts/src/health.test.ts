import { describe, expect, it } from 'vitest'
import { openSettings, SETTINGS_OPEN_COMMAND } from './health'

describe('openSettings', () => {
  it('opens the settings of the plugin at the setting of the key', () => {
    expect(openSettings('harbor-key')).toStrictEqual({
      command: SETTINGS_OPEN_COMMAND,
      values: { setting: 'harbor-key' },
    })
  })

  it('opens the settings of the plugin without a key', () => {
    expect(openSettings()).toStrictEqual({ command: SETTINGS_OPEN_COMMAND, values: {} })
  })
})

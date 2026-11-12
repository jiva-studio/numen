/** The settings as when conditions read them, `config.<dotted key>`, and the conditions of the settings. */
import { describe, expect, it } from 'vitest'
import type { PluginSettingInfo } from '@/entities/settings'
import { getSettingWhen, getSettingsConfig } from './settingsConfig'

const setting = (key: string, value: unknown, when?: PluginSettingInfo['when']) =>
  ({ key, value, ...(when ? { when } : {}) }) as unknown as PluginSettingInfo

const OWNERS = [
  { id: 'app', name: '', settings: [setting('appearance.theme', 'dark')] },
  {
    id: 'org.example.ledger',
    name: 'Ledger',
    settings: [
      setting('mode', 'custom'),
      setting('path', '/tmp', { setting: 'mode', is: ['custom', 'local'] }),
      setting('depth', 2, { setting: 'path', is: ['/tmp'] }),
    ],
  },
]

describe('the settings as config', () => {
  it('hold every setting by its dotted key, the application under app and a plugin under its id', () => {
    expect(getSettingsConfig(OWNERS)).toStrictEqual({
      app: { appearance: { theme: 'dark' } },
      org: { example: { ledger: { mode: 'custom', path: '/tmp', depth: 2 } } },
    })
  })
})

describe('the condition of a setting command', () => {
  it('is empty for a setting shown always', () => {
    expect(getSettingWhen(OWNERS[1]!, OWNERS[1]!.settings[0]!)).toBe('')
  })

  it('reads the setting its condition names, and the conditions of that setting', () => {
    expect(getSettingWhen(OWNERS[1]!, OWNERS[1]!.settings[2]!)).toBe(
      "string(config.org.example.ledger.path) in ['/tmp'] && string(config.org.example.ledger.mode) in ['custom', 'local']",
    )
  })
})

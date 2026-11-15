/** The settings as when conditions read them, `config.<dotted key>`, and the conditions of the settings. */
import { describe, expect, it } from 'vitest'
import type { PluginSettingInfo } from '@/entities/settings'
import { getSettingWhen, getSettingsConfig } from './settingsConfig'

const setting = (key: string, value: unknown, when?: PluginSettingInfo['when']) =>
  ({ key, value, ...(when ? { when } : {}) }) as never

const OWNERS = [
  { id: 'app', name: '', settings: [setting('appearance.theme', 'dark'), setting('unset', undefined)] },
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
    const owner = OWNERS[1]
    const first = owner?.settings[0]
    expect(owner && first ? getSettingWhen(owner, first) : null).toBe('')
  })

  it('reads the setting its condition names, and the conditions of that setting', () => {
    const owner = OWNERS[1]
    const third = owner?.settings[2]
    expect(owner && third ? getSettingWhen(owner, third) : null).toBe(
      "string(config.org.example.ledger.path) in ['/tmp'] && string(config.org.example.ledger.mode) in ['custom', 'local']",
    )
  })

  it('escapes quotes, stops on missing parent settings, and breaks cycles in conditions', () => {
    const cyclicOwner = {
      id: 'org.example.cycle',
      name: 'Cycle',
      settings: [
        setting('a', '1', { setting: 'b', is: ["don't"] }),
        setting('b', '2', { setting: 'a', is: ['yes'] }),
        setting('orphan', '3', { setting: 'missing', is: ['no'] }),
      ],
    }
    const [first, , orphan] = cyclicOwner.settings

    expect(first ? getSettingWhen(cyclicOwner, first) : null).toBe(
      "string(config.org.example.cycle.b) in ['don\\'t'] && string(config.org.example.cycle.a) in ['yes']",
    )
    expect(orphan ? getSettingWhen(cyclicOwner, orphan) : null).toBe(
      "string(config.org.example.cycle.missing) in ['no']",
    )
  })
})
